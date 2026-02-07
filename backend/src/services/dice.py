"""CoC 7e Dice Engine Service."""
from dataclasses import dataclass
from enum import Enum
from random import randint, seed as seed_random
from typing import Optional


class SuccessLevel(Enum):
    """Success level for skill checks."""

    EXTREME_SUCCESS = "extreme_success"
    HARD_SUCCESS = "hard_success"
    REGULAR_SUCCESS = "regular_success"
    FAILURE = "failure"


class BonusPenalty(Enum):
    """Bonus/penalty dice levels.

    In CoC 7e:
    - REGULAR: Roll 1d100, compare to skill
    - HARD: Roll with bonus dice (2d100, take lower)
    - EXTREME: Roll with double bonus (3d100, take lowest)
    - ONE_STEP_BONUS: Move one step toward easier difficulty
    - ONE_STEP_PENALTY: Move one step toward harder difficulty
    """

    REGULAR = "regular"
    HARD = "hard"
    EXTREME = "extreme"
    ONE_STEP_BONUS = "one_step_bonus"
    ONE_STEP_PENALTY = "one_step_penalty"


@dataclass
class RollResult:
    """Result of a dice roll."""

    value: int
    success_level: SuccessLevel
    raw_rolls: Optional[list[int]] = None
    bonus_penalty: Optional[BonusPenalty] = None


def roll_d100(seed: Optional[int] = None) -> int:
    """Roll a single d100 (1-100).

    Args:
        seed: Optional random seed for reproducibility.

    Returns:
        Random integer between 1 and 100.
    """
    if seed is not None:
        seed_random(seed)
    return randint(1, 100)


def roll_multiple(count: int, seed: Optional[int] = None) -> list[int]:
    """Roll multiple d100 dice.

    Args:
        count: Number of dice to roll.
        seed: Optional random seed for reproducibility.

    Returns:
        List of random integers between 1 and 100.
    """
    if seed is not None:
        seed_random(seed)
    return [randint(1, 100) for _ in range(count)]


def determine_success(roll: int, skill: int) -> SuccessLevel:
    """Determine success level from a roll and skill value.

    CoC 7e success levels:
    - Extreme success: roll <= skill / 5
    - Hard success: roll <= skill / 2
    - Regular success: roll <= skill
    - Failure: roll > skill
    - Critical: roll of 1 is always extreme success
    - Fumble: roll of 100 is always failure

    Args:
        roll: The dice roll (1-100).
        skill: The skill or attribute value.

    Returns:
        The success level.
    """
    # Critical and fumble override
    if roll == 1:
        return SuccessLevel.EXTREME_SUCCESS
    if roll == 100:
        return SuccessLevel.FAILURE

    # Calculate thresholds
    extreme_threshold = max(1, skill // 5)
    hard_threshold = skill // 2

    # Determine success level
    if roll <= extreme_threshold:
        return SuccessLevel.EXTREME_SUCCESS
    elif roll <= hard_threshold:
        return SuccessLevel.HARD_SUCCESS
    elif roll <= skill:
        return SuccessLevel.REGULAR_SUCCESS
    else:
        return SuccessLevel.FAILURE


def apply_bonus_penalty(
    skill: int, bonus_penalty: BonusPenalty
) -> tuple[int, BonusPenalty]:
    """Apply bonus/penalty to difficulty.

    Args:
        skill: The base skill value.
        bonus_penalty: The bonus/penalty level.

    Returns:
        Tuple of (adjusted skill, effective difficulty).
    """
    base_skill = skill

    match bonus_penalty:
        case BonusPenalty.EXTREME:
            # Extreme difficulty: harder (skill / 2)
            return base_skill // 2, BonusPenalty.EXTREME
        case BonusPenalty.HARD:
            # Hard difficulty: slightly harder (skill * 2 / 3)
            return (base_skill * 2) // 3, BonusPenalty.HARD
        case BonusPenalty.REGULAR:
            # Regular difficulty
            return base_skill, BonusPenalty.REGULAR
        case BonusPenalty.ONE_STEP_BONUS:
            # One step easier: skill * 1.5
            return (base_skill * 3) // 2, BonusPenalty.ONE_STEP_BONUS
        case BonusPenalty.ONE_STEP_PENALTY:
            # One step harder: skill * 2 / 3
            return (base_skill * 2) // 3, BonusPenalty.ONE_STEP_PENALTY


def roll_check(
    skill: int,
    roll: Optional[int] = None,
    bonus_penalty: BonusPenalty = BonusPenalty.REGULAR,
    seed: Optional[int] = None,
) -> RollResult:
    """Perform a skill check with optional bonus/penalty dice.

    Args:
        skill: The skill or attribute value to check against.
        roll: Optional specific roll value (for testing).
        bonus_penalty: Bonus/penalty level affecting difficulty.
        seed: Optional random seed for reproducibility.

    Returns:
        RollResult with value, success level, and metadata.
    """
    # Apply bonus/penalty to get effective difficulty
    effective_skill, effective_difficulty = apply_bonus_penalty(skill, bonus_penalty)

    # Roll the dice
    if roll is not None:
        actual_roll = roll
        raw_rolls = [roll]
    else:
        # Roll based on difficulty
        match bonus_penalty:
            case BonusPenalty.EXTREME:
                # Roll 3d100, take lowest
                raw_rolls = roll_multiple(3, seed)
                actual_roll = min(raw_rolls)
            case BonusPenalty.HARD:
                # Roll 2d100, take lower
                raw_rolls = roll_multiple(2, seed)
                actual_roll = min(raw_rolls)
            case BonusPenalty.REGULAR | BonusPenalty.ONE_STEP_BONUS | BonusPenalty.ONE_STEP_PENALTY:
                # Roll 1d100
                actual_roll = roll_d100(seed)
                raw_rolls = [actual_roll]

    # Determine success level based on effective skill
    success_level = determine_success(actual_roll, effective_skill)

    return RollResult(
        value=actual_roll,
        success_level=success_level,
        raw_rolls=raw_rolls,
        bonus_penalty=effective_difficulty,
    )
