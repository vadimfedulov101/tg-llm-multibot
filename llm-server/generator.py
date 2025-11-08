"""LLM Text Generation Base Module.

Generator superclass orchestrates dialog processing, token management, and
LLM inference for specialized Responder/Selector subclasses.

Key Components:
- Dialog Processing:   Normalization and string convertion
- Token Management:    Mode-based configuration (respond/rate)
- Generation Pipeline: Prompt templating and stop sequence utilization
"""

import re
from typing import Optional

import torch

from models import LLM, Settings
from stopper import get_stop_vars, trim_stop_sequences


class Generator:
    """Superclass for Responder and Selector classes.

    self.generate(user_prompt: str) to generate LLM response.
    """

    SYSTEM_TEMPLATE = "<|start_header_id|>system<|end_header_id|>%s<|eot_id|>"
    USER_TEMPLATE = "<|start_header_id|>user<|end_header_id|>%s<|eot_id|>"
    ASSISTANT_TEMPLATE = "<|start_header_id|>assistant<|end_header_id|>"

    RESP_TOKEN_ERR_MSG = "No token number specified for response."
    MODE_ERR_MSG = "Mode is %s. Set to 'respond' or 'rate'!"
    RATE_TOKEN_ERR_MSG = "No token number for rate."

    def __init__(
        self,
        llm: LLM,
        settings: Settings,
        chat_context: list[str],
        reply_chain: list[str],
        responses: Optional[list[str]],
    ) -> None:
        self._llm = llm
        self._settings = settings
        self._chat_context = self.normalize_text(chat_context)
        self._chat_context_str = "\n".join(chat_context)
        self._reply_chain = self.normalize_text(reply_chain)
        self._reply_chain_str = "\n".join(reply_chain)

        self._set_stopping_criteria()
        self._set_token_num()

        self.responses = responses

    @staticmethod
    def normalize_text(text: list[str]) -> list[str]:
        for i, msg in enumerate(text):
            if "\n" in msg:
                text[i] = re.sub("\n+", r"\\n", msg)
        return text

    def _set_stopping_criteria(self) -> None:
        tokenizer = self._llm.tokenizer
        reply_chain = self._reply_chain

        stopping_criteria, stop_token_ids = get_stop_vars(tokenizer,
                                                          reply_chain)
        self.stopping_criteria = stopping_criteria
        self.stop_token_ids = stop_token_ids

    def _set_token_num(self) -> None:
        """Set self.max_new_tokens token number based on mode and values.

        select mode: rate_tokens* ?
        respond mode: response_tokens*/inputs.size(1) + response_token_shift ?
        * Must be non-zero; ? Raise error on failure

        Args:
            self

        Returns: None

        Raises
        ------
            ValueError: problem

        """
        tokenizer = self._llm.tokenizer
        mode = self._llm.mode
        settings = self._settings
        reply_chain = self._reply_chain

        match mode:
            case "rate":
                rate_tokens = settings.rate_tokens
                # Set static (specified)
                if rate_tokens != 0:
                    self.max_new_tokens = settings.rate_tokens
                else:
                    err = ValueError(self.RATE_TOKEN_ERR_MSG)
                    raise err
            case "response":
                response_tokens = settings.response_tokens
                response_token_shift = settings.response_token_shift
                # Set static
                if settings.response_tokens != 0:
                    self.max_new_tokens = response_tokens
                # Set with shift
                elif settings.response_token_shift != 0:
                    inputs = tokenizer.encode(
                        reply_chain[-1], return_tensors="pt")
                    self.max_new_tokens = inputs.size(1) + response_token_shift
                else:
                    err = ValueError(self.RESP_TOKEN_ERR_MSG)
                    raise err
            case _:
                err = ValueError(self.MODE_ERR_MSG % mode)
                raise err

    def _new_prompt(self, user_prompt: str) -> str:
        settings = self._settings

        system_prompt = settings.system_prompt

        prompt = self.SYSTEM_TEMPLATE % system_prompt
        prompt += self.USER_TEMPLATE % user_prompt
        prompt += self.ASSISTANT_TEMPLATE

        return prompt

    def generate(self, user_prompt: str) -> str:
        model = self._llm.model
        tokenizer = self._llm.tokenizer
        settings = self._settings

        max_new_tokens = self.max_new_tokens
        stop_token_ids = self.stop_token_ids
        stopping_criteria = self.stopping_criteria

        prompt = self._new_prompt(user_prompt)
        inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
        with torch.no_grad():
            output_ids = model.generate(
                **inputs,
                do_sample=True,
                bos_token_id=tokenizer.bos_token_id,
                eos_token_id=tokenizer.eos_token_id,
                pad_token_id=tokenizer.pad_token_id,
                temperature=settings.temperature,
                repetition_penalty=settings.repetition_penalty,
                top_p=settings.top_p,
                top_k=settings.top_k,
                max_new_tokens=max_new_tokens,
                stopping_criteria=stopping_criteria,
            )
            processed_ids = trim_stop_sequences(output_ids, stop_token_ids)
            response_raw = tokenizer.batch_decode(
                processed_ids, skip_special_tokens=True
            )[0]
            response_raw = response_raw.split("assistant")[-1].strip()

        return response_raw
