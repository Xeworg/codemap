"""Parsers module for CodeMap.

Provides code parsing functionality for multiple programming languages.
Supports AST-based analysis using tree-sitter for language-agnostic parsing.

Available Parsers
-----------------
- TreeSitterParser: Multi-language parser using tree-sitter
- BaseASTParser: Abstract base class for custom parsers

Usage
-----
>>> from codemap.parsers import TreeSitterParser
>>> parser = TreeSitterParser()
>>> result = parser.parse_file(Path("example.py"))
>>> for entity in result.entities:
...     print(f"{entity.type}: {entity.name}")
"""

from codemap.parsers.ast_parser import (
    BaseASTParser,
    Entity,
    CallEdge,
    Dependency,
    ParseResult,
)
from codemap.parsers.tree_sitter_parser import TreeSitterParser

__all__ = [
    "BaseASTParser",
    "Entity",
    "CallEdge",
    "Dependency",
    "ParseResult",
    "TreeSitterParser",
]
