"""create characters table

Revision ID: 002
Revises: 0e7427814ae0
Create Date: 2026-02-07

"""
from typing import Union
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision: str = '002'
down_revision: Union[str, None] = '0e7427814ae0'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Upgrade schema - create characters table."""
    # Create enum type if it doesn't exist
    op.execute("DROP TYPE IF EXISTS charactertype CASCADE")
    op.execute("CREATE TYPE charactertype AS ENUM ('player', 'npc')")

    # Create characters table
    op.create_table(
        'characters',
        sa.Column('character_id', sa.String(), nullable=False),
        sa.Column('player_id', sa.String(), nullable=True),
        sa.Column('name', sa.String(), nullable=False),
        sa.Column('type', sa.Enum('player', 'npc', name='charactertype'), nullable=False),
        sa.Column('core_attributes', sa.JSON(), nullable=False),
        sa.Column('derived_attributes', sa.JSON(), nullable=False),
        sa.Column('skills', sa.JSON(), nullable=False),
        sa.Column('inventory', sa.JSON(), default=list),
        sa.Column('clues', sa.JSON(), default=list),
        sa.Column('status', sa.JSON(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['player_id'], ['users.user_id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('character_id')
    )

    # Create indexes
    op.create_index('ix_characters_character_id', 'characters', ['character_id'], unique=False)
    op.create_index('ix_characters_player_id', 'characters', ['player_id'], unique=False)
    op.create_index('ix_characters_name', 'characters', ['name'], unique=False)


def downgrade() -> None:
    """Downgrade schema - drop characters table."""
    op.drop_index('ix_characters_name', table_name='characters')
    op.drop_index('ix_characters_player_id', table_name='characters')
    op.drop_index('ix_characters_character_id', table_name='characters')
    op.drop_table('characters')
    op.execute("DROP TYPE IF EXISTS charactertype CASCADE")
