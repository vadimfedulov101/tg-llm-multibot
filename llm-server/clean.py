"""LLM Output Sanitization Module.

Clean module implements multi-stage text normalization for raw LLM output,
enforcing formatting rules and grammatical correctness via regex-processing.

Key Components:
- EOS Handling:    Stripping of verbalized end-of-sequence markers (EOS)
- Role Removal:    Speaker identifier and preceding context elimination
- Structurization: Enforcing complete sentences and proper capitalization
- Spacing Fix:     Punctuation spacing and line breaks standartization

Implementation Notes:
- Order Sensitive:   Dependent on the order of applied operations on raw output
- Idempotent Design: Safe for repeated application on same text
"""

import re


def _strip_verbalized_eos(text: str) -> str:
    text = text.split("EOS")[0]
    return text


def _strip_extra_names(text: str) -> str:
    """Strip regex-matches of role identifyers."""
    text = re.sub(r"^.{0,24}:\s*", "", text)

    return text


def _strip_incomplete_sentences(text: str) -> str:
    """Strip everything after last valid sentence punctuation sign (.!?)."""
    if text:
        text = text[0].upper() + text[1:]

    last_index = max((text.rfind(char) for char in ".!?"), default=-1)
    if last_index != -1:
        text = text[: last_index + 1]

    return text


def _fix_punctuation(text: str) -> str:
    """Adjust all punctuation signs according to the norms."""
    # Dots
    text = re.sub(r"\s*\.", ".", text)
    # Commas
    text = re.sub(r"\s*,\s*", ", ", text)

    # Spaces
    pattern = r"\s*([.!?]\s*)(\S)"

    def fix(m: re.Match[str]) -> str:
        punct = m.group(1)
        letter = m.group(2)
        return f"{punct.strip()} {letter.upper()}"

    text = re.sub(pattern, fix, text)

    return text


def _sub_custom_n(text: str) -> str:
    r"""Substitute escaped \\n with new line character."""
    text = re.sub(r"\\n", "\n", text)
    return text


def clean(text: str) -> str:
    """Clean LLM generated text."""
    text = _strip_verbalized_eos(text)
    text = _strip_extra_names(text)
    text = _strip_incomplete_sentences(text)

    text = _fix_punctuation(text)
    text = _sub_custom_n(text)

    return text
