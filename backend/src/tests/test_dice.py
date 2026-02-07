"""Tests for dice engine service."""
import pytest

from src.services.dice import (
    roll_d100,
    RollResult,
    SuccessLevel,
    roll_check,
    BonusPenalty,
)


class TestRollD100:
    """Test basic d100 rolling."""

    def test_roll_d100_returns_int(self):
        """Roll should return an integer."""
        result = roll_d100()
        assert isinstance(result, int)

    def test_roll_d100_range(self):
        """Roll should be between 1 and 100 inclusive."""
        for _ in range(100):
            result = roll_d100()
            assert 1 <= result <= 100

    def test_roll_d100_with_seed(self):
        """Roll with same seed should return same result."""
        result1 = roll_d100(seed=42)
        result2 = roll_d100(seed=42)
        assert result1 == result2


class TestRollResult:
    """Test RollResult dataclass."""

    def test_roll_result_creation(self):
        """Should create RollResult with value and success level."""
        result = RollResult(value=50, success_level=SuccessLevel.REGULAR_SUCCESS)
        assert result.value == 50
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS


class TestSuccessLevel:
    """Test SuccessLevel enum."""

    def test_success_level_values(self):
        """SuccessLevel should have expected values."""
        assert SuccessLevel.EXTREME_SUCCESS.value == "extreme_success"
        assert SuccessLevel.HARD_SUCCESS.value == "hard_success"
        assert SuccessLevel.REGULAR_SUCCESS.value == "regular_success"
        assert SuccessLevel.FAILURE.value == "failure"


class TestRollCheck:
    """Test skill check with d100."""

    def test_regular_success_on_half(self):
        """Rolling at or below skill should be regular success."""
        # At 50 skill, rolling 50 should be regular success
        result = roll_check(skill=50, roll=50)
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS

    def test_regular_success_below_skill(self):
        """Rolling below skill should be regular success."""
        result = roll_check(skill=50, roll=49)
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS

    def test_failure_above_skill(self):
        """Rolling above skill should be failure."""
        result = roll_check(skill=50, roll=51)
        assert result.success_level == SuccessLevel.FAILURE

    def test_hard_success_at_half(self):
        """Rolling at or below half skill should be hard success."""
        result = roll_check(skill=50, roll=25)
        assert result.success_level == SuccessLevel.HARD_SUCCESS

    def test_extreme_success_at_fifth(self):
        """Rolling at or below one-fifth skill should be extreme success."""
        result = roll_check(skill=50, roll=10)
        assert result.success_level == SuccessLevel.EXTREME_SUCCESS

    def test_fumble_on_1(self):
        """Rolling 1 should be extreme success regardless of skill."""
        result = roll_check(skill=10, roll=1)
        assert result.success_level == SuccessLevel.EXTREME_SUCCESS

    def test_critical_failure_on_100(self):
        """Rolling 100 should be failure regardless of skill."""
        result = roll_check(skill=99, roll=100)
        assert result.success_level == SuccessLevel.FAILURE


class TestBonusPenaltyDice:
    """Test bonus and penalty dice mechanics."""

    def test_bonus_dice_one_step(self):
        """Bonus dice at regular difficulty."""
        # With bonus dice, we roll two and take lower
        result = roll_check(skill=50, roll=60, bonus_penalty=BonusPenalty.ONE_STEP_BONUS)
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS

    def test_penalty_dice_one_step(self):
        """Penalty dice at regular difficulty."""
        # With penalty dice, we roll two and take higher
        result = roll_check(skill=50, roll=40, bonus_penalty=BonusPenalty.ONE_STEP_PENALTY)
        assert result.success_level == SuccessLevel.FAILURE

    def test_bonus_dice_regular_difficulty(self):
        """Bonus dice at regular difficulty."""
        result = roll_check(skill=50, bonus_penalty=BonusPenalty.REGULAR)
        assert isinstance(result, RollResult)

    def test_bonus_diece_hard_difficulty(self):
        """Bonus dice at hard difficulty."""
        result = roll_check(skill=50, bonus_penalty=BonusPenalty.HARD)
        assert isinstance(result, RollResult)

    def test_bonus_dice_extreme_difficulty(self):
        """Bonus dice at extreme difficulty."""
        result = roll_check(skill=50, bonus_penalty=BonusPenalty.EXTREME)
        assert isinstance(result, RollResult)


class TestDiceEngine:
    """Integration tests for dice engine."""

    def test_multiple_rolls_distribution(self):
        """Multiple rolls should have reasonable distribution."""
        results = [roll_d100() for _ in range(1000)]
        # Mean should be around 50.5
        mean = sum(results) / len(results)
        assert 40 < mean < 60

    def test_character_integration(self):
        """Test dice engine with character stats."""
        # A character with 70 luck rolling under luck should succeed
        result = roll_check(skill=70, roll=70)
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS

        # Same character rolling 71 should fail
        result = roll_check(skill=70, roll=71)
        assert result.success_level == SuccessLevel.FAILURE
