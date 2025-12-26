"""Módulo de parsers para CodeMap.

Proporciona funcionalidad de análisis de código para múltiples lenguajes de programación.
Soporta análisis basado en AST usando tree-sitter para parsing agnóstico del lenguaje.

Parsers Disponibles
-------------------
- TreeSitterParser: Parser multi-lenguaje usando tree-sitter
- BaseASTParser: Clase base abstracta para parsers personalizados

Uso
---
>>> from codemap.parsers import TreeSitterParser
>>> parser = TreeSitterParser()
>>> resultado = parser.parse_file(Path("ejemplo.py"))
>>> for entidad in resultado.entidades:
...     print(f"{entidad.tipo}: {entidad.nombre}")
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
