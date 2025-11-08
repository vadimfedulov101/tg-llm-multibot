"""Chain of Thought Response Generation Module.

Responder class manages parallelized Chain of Thought reasoning for response
generation with cleaning, validation, retries and batch processing.

Key Components:
- Responder:      Orchestration of CoT workflow with colored console feedback
- Thought Chains: Parallel reasoning paths managed via NumPy arrays
- Clean/Validate: Imported validation and cleaning pipelines for LLM output
"""

from contextlib import contextmanager

import numpy as np
from colorama import Fore, init
from numpy.typing import NDArray

from clean import clean
from generator import Generator
from validate import validate

init(autoreset=True)


class Responder(Generator):
    """Responder initialized with Generator initialize method.

    self.respond() to respond based on passed dialog.
    """

    SUCCESS_MSG = Fore.GREEN + "[Success]"
    FAILURE_MSG = Fore.RED + "[Failure]"
    OVERFLOW_MSG = Fore.RED + "[Fatal] Max attempts exceeded. Generation skip."
    FATAL_MSG = "No thoughts generated with prompt: %s."

    def _think(self, chain_prompt: str, thought_chain: NDArray) -> list[str]:
        """Generate thought based on chain prompt with thought chain."""
        thoughts = []

        settings = self._settings
        batch_size = settings.response_batch_size
        max_attempts = batch_size * 3
        verbose = self._llm.verbose

        chat_context = self._chat_context
        chat_context_str = self._chat_context_str

        reply_chain = self._reply_chain
        reply_chain_str = self._reply_chain_str

        user_prompt = chain_prompt.format(
            chat_context_str, reply_chain_str, *thought_chain)
        if verbose:
            print(user_prompt)

        for attempt in range(max_attempts):
            current_try = attempt + 1
            print(f"Try {current_try:02}:", end=" ")

            thought_raw = self.generate(user_prompt)
            thought = clean(thought_raw)

            validation_context = chat_context + reply_chain
            ok, err = validate(thought, chain_prompt, validation_context)
            if ok:
                print(self.SUCCESS_MSG)
                thoughts.append(thought)
                if verbose:
                    print(thought_raw)
                if len(thoughts) >= batch_size:
                    break
            else:
                print(self.FAILURE_MSG, end=" ")
                print(err)

        if len(thoughts) < batch_size:
            print(self.OVERFLOW_MSG)

        if len(thoughts) == 0:
            raise ValueError(self.FATAL_MSG % chain_prompt)

        return thoughts

    @contextmanager
    def _temp_batch_size(self, new_size: int) -> None:
        """Temporarily modify the batch size."""
        original = self._settings.response_batch_size
        self._settings.response_batch_size = new_size
        try:
            yield
        finally:
            self._settings.response_batch_size = original

    def respond(self) -> list[str]:
        """Implement Chain of Thought algorithm."""
        settings = self._settings
        chain_prompts = settings.chain_prompts
        batch_size = settings.response_batch_size

        # Generate initial thoughts for all parallel chains
        print("Step 1: CoT start")
        chains = [[] for _ in range(batch_size)]  # Each chain starts empty
        initial_thoughts = self._think(chain_prompts[0], np.array([]))

        # Distribute initial thoughts to chains
        for chain_idx, thought in enumerate(initial_thoughts):
            if chain_idx < batch_size:
                chains[chain_idx].append(thought)

        if len(chain_prompts) == 1:
            return [chain[-1] for chain in chains if chain]

        # Continue each chain through remaining steps
        print("Step 2: CoT continue")
        with self._temp_batch_size(1):
            for step_idx, chain_prompt in enumerate(chain_prompts[1:], 1):
                for chain_idx, chain in enumerate(chains):
                    if len(chain) >= step_idx:  # Chain exists and is ready
                        new_thoughts = self._think(
                            chain_prompt, np.array(chain))
                        if new_thoughts:
                            chains[chain_idx].append(new_thoughts[0])

        # Return the final thought from each chain
        return [chain[-1] for chain in chains if chain]
