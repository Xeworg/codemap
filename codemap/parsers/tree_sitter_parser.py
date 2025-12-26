"""Tree-sitter parser wrapper for CodeMap.

Provides multi-language parsing capabilities using tree-sitter.
Supports Python, JavaScript, TypeScript, Java, Go, Rust, C++, Ruby, and PHP.

Features
--------
- Language detection from file extension
- Entity extraction (classes, functions, methods)
- Call graph construction
- Dependency analysis
- Basic metrics calculation

Example
-------
>>> from pathlib import Path
>>> from codemap.parsers import TreeSitterParser
>>> parser = TreeSitterParser()
>>> result = parser.parse_file(Path("example.py"))
>>> for entity in result.entities:
...     print(f"{entity.type}: {entity.name}")

Dependencies
------------
Requires tree-sitter and tree-sitter-libraries packages:
    pip install tree-sitter tree-sitter-languages
"""

from pathlib import Path
from typing import List, Optional, Dict, Any
from dataclasses import dataclass, field
import json

try:
    import tree_sitter
    from tree_sitter import Language, Parser

    TREE_SITTER_AVAILABLE = True
except ImportError:
    TREE_SITTER_AVAILABLE = False

from codemap.parsers.ast_parser import (
    BaseASTParser,
    Entity,
    CallEdge,
    Dependency,
    ParseResult,
)
from codemap.parsers import utils as parser_utils


@dataclass
class TreeSitterEntity(Entity):
    """Entity extracted from tree-sitter AST.

    Extends Entity with tree-sitter-specific fields.

    Attributes:
        node_type: Raw tree-sitter node type (e.g., 'function_definition')
        is_definition: True if this is a definition node
    """

    node_type: str = ""
    is_definition: bool = False


class TreeSitterParser(BaseASTParser):
    """Multi-language parser using tree-sitter.

    Parses source code files and extracts entities, calls, and dependencies
    using tree-sitter's language-agnostic AST representation.

    Supported Languages
    -------------------
    - Python (.py)
    - JavaScript (.js, .jsx)
    - TypeScript (.ts, .tsx)
    - Java (.java)
    - Go (.go)
    - Rust (.rs)
    - C/C++ (.c, .cpp, .h)
    - Ruby (.rb)
    - PHP (.php)

    Attributes:
        parsers: Dictionary mapping language names to tree-sitter parsers.

    Example:
        >>> parser = TreeSitterParser()
        >>> if parser.is_available():
        ...     result = parser.parse_file(Path("main.py"))
        ...     print(f"Found {len(result.entities)} entities")
    """

    LANGUAGE_CONFIG: Dict[str, str] = {
        "python": "python",
        "javascript": "javascript",
        "typescript": "typescript",
        "java": "java",
        "go": "go",
        "rust": "rust",
        "cpp": "cpp",
        "c": "c",
        "ruby": "ruby",
        "php": "php",
    }

    DEFINITION_TYPES = {
        "function_definition",
        "class_definition",
        "method_definition",
        "function_declaration",
        "class_declaration",
    }

    CALL_TYPES = {
        "call_expression",
        "method_call",
        "identifier",
    }

    IMPORT_TYPES = {
        "import_statement",
        "import_from_statement",
        "require_call",
    }

    def __init__(self):
        """Initialize parser and load language parsers."""
        super().__init__()
        self.parsers: Dict[str, Parser] = {}
        self._load_languages()

    def _load_languages(self):
        """Load tree-sitter language parsers.

        Attempts to load all configured language parsers using
        tree_sitter_languages for efficient parser retrieval.
        """
        if not TREE_SITTER_AVAILABLE:
            return

        try:
            from tree_sitter_languages import get_language, get_parser

            for lang, _ in self.LANGUAGE_CONFIG.items():
                self.parsers[lang] = get_parser(lang)
        except Exception:
            pass

    def is_available(self) -> bool:
        """Check if tree-sitter is properly installed and configured.

        Returns:
            True if tree-sitter is available and at least one language
            parser is loaded.
        """
        return TREE_SITTER_AVAILABLE and len(self.parsers) > 0

    def parse_file(self, file_path: Path) -> ParseResult:
        """Parse a file using tree-sitter.

        Args:
            file_path: Path to the file to parse.

        Returns:
            ParseResult containing extracted entities, calls, dependencies,
            and metrics.
        """
        content = self.read_file(file_path)
        if content is None:
            return ParseResult(
                file=str(file_path), language="unknown", errors=["Could not read file"]
            )
        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Parse file content using tree-sitter.

        Args:
            content: Raw source code content.
            file_path: Path object for reference (language detection).

        Returns:
            ParseResult containing extracted entities, calls, dependencies,
            and metrics.
        """
        result = ParseResult(
            file=str(file_path),
            language=parser_utils.detect_language(file_path) or "unknown",
        )

        if not TREE_SITTER_AVAILABLE:
            result.errors.append("tree-sitter not available")
            return result

        parser = self.parsers.get(result.language)
        if parser is None:
            result.errors.append(f"No parser for language: {result.language}")
            return result

        try:
            tree = parser.parse(bytes(content, "utf-8"))
            result.entities = self._extract_entities(tree.root_node, content, file_path)
            result.calls = self._extract_calls(tree.root_node, content)
            result.dependencies = self._extract_dependencies(tree.root_node, content)
            result.metrics = self._compute_metrics(content, result.entities)
        except Exception as e:
            result.errors.append(str(e))

        return result

    def _extract_entities(
        self, node, content: str, file_path: Path
    ) -> List[TreeSitterEntity]:
        """Extract entity definitions from AST.

        Recursively traverses the AST to find all definition nodes
        (functions, classes, methods).

        Args:
            node: Current tree-sitter node.
            content: Source code content for line extraction.
            file_path: File path for entity metadata.

        Returns:
            List of extracted TreeSitterEntity objects.
        """
        entities = []
        lines = content.split("\n")

        for child in node.children:
            if child.type in self.DEFINITION_TYPES:
                entity = self._node_to_entity(child, content, file_path)
                if entity:
                    entities.append(entity)

            entities.extend(self._extract_entities(child, content, file_path))

        return entities

    def _node_to_entity(
        self, node, content: str, file_path: Path
    ) -> Optional[TreeSitterEntity]:
        """Convert a tree-sitter node to an Entity.

        Args:
            node: Tree-sitter node to convert.
            content: Source code content.
            file_path: File path for entity metadata.

        Returns:
            TreeSitterEntity if conversion successful, None otherwise.
        """
        name = self._extract_name(node, content)
        if not name:
            return None

        line = node.start_point[0] + 1
        end_line = node.end_point[0] + 1
        loc = end_line - line

        entity_type = self._map_node_type(node.type)

        entity_id = self.generate_entity_id(entity_type, name)

        docstring = self._extract_docstring_from_node(node, content)

        return TreeSitterEntity(
            id=entity_id,
            type=entity_type,
            name=name,
            file=str(file_path),
            line=line,
            end_line=end_line,
            loc=loc,
            docstring=docstring,
            node_type=node.type,
            is_definition=True,
        )

    def _extract_name(self, node, content: str) -> Optional[str]:
        """Extract entity name from node.

        Args:
            node: Tree-sitter node.
            content: Source code content.

        Returns:
            Entity name if found, None otherwise.
        """
        lines = content.split("\n")

        for child in node.children:
            if child.type == "identifier":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()
            if child.type == "name":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()

        return None

    def _map_node_type(self, node_type: str) -> str:
        """Map tree-sitter node types to entity types.

        Args:
            node_type: Raw tree-sitter node type string.

        Returns:
            Mapped entity type (class, function, method, etc.)
        """
        type_map = {
            "function_definition": "function",
            "function_declaration": "function",
            "class_definition": "class",
            "class_declaration": "class",
            "method_definition": "method",
            "method_declaration": "method",
        }
        return type_map.get(node_type, "function")

    def _extract_docstring_from_node(self, node, content: str) -> Optional[str]:
        """Extract docstring from node.

        Args:
            node: Tree-sitter node.
            content: Source code content.

        Returns:
            Docstring text if found, None otherwise.
        """
        lines = content.split("\n")
        start_line = node.start_point[0]

        if start_line + 1 < len(lines):
            first_line = lines[start_line].strip()
            if first_line.startswith('"""') or first_line.startswith("'''"):
                end_char = first_line[3:]
                if end_char and not end_char.endswith("\\"):
                    return first_line[3:-3]

        return None

    def _extract_calls(self, node, content: str) -> List[CallEdge]:
        """Extract function calls from AST.

        Args:
            node: Current tree-sitter node.
            content: Source code content.

        Returns:
            List of CallEdge objects representing function calls.
        """
        calls = []

        for child in node.children:
            if child.type in self.CALL_TYPES:
                call = self._node_to_call(child, content)
                if call:
                    calls.append(call)

            calls.extend(self._extract_calls(child, content))

        return calls

    def _node_to_call(self, node, content: str) -> Optional[CallEdge]:
        """Convert a call node to a CallEdge.

        Args:
            node: Tree-sitter node representing a call.
            content: Source code content.

        Returns:
            CallEdge if conversion successful, None otherwise.
        """
        lines = content.split("\n")
        line = node.start_point[0] + 1

        if node.children:
            callee = node.children[0]
            if callee.type == "identifier":
                line_idx = callee.start_point[0]
                if line_idx < len(lines):
                    callee_name = lines[line_idx].strip()
                    return CallEdge(
                        from_id="",
                        to_id=f"func:{callee_name}",
                        line=line,
                    )

        return None

    def _extract_dependencies(self, node, content: str) -> List[Dependency]:
        """Extract module dependencies from AST.

        Args:
            node: Current tree-sitter node.
            content: Source code content.

        Returns:
            List of Dependency objects.
        """
        deps = []

        for child in node.children:
            if child.type in self.IMPORT_TYPES:
                dep = self._node_to_dependency(child, content)
                if dep:
                    deps.append(dep)

            deps.extend(self._extract_dependencies(child, content))

        return deps

    def _node_to_dependency(self, node, content: str) -> Optional[Dependency]:
        """Convert import node to Dependency.

        Args:
            node: Tree-sitter node representing an import.
            content: Source code content.

        Returns:
            Dependency if conversion successful, None otherwise.
        """
        lines = content.split("\n")

        for child in node.children:
            if child.type == "string":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    module = lines[line_idx].strip().strip('"').strip("'")
                    return Dependency(
                        source_module="",
                        target_module=module,
                        dependency_type="import",
                    )

        return None

    def _compute_metrics(self, content: str, entities: List[Entity]) -> Dict[str, Any]:
        """Compute metrics for the parsed content.

        Args:
            content: Source code content.
            entities: List of extracted entities.

        Returns:
            Dictionary containing metrics:
            - total_loc: Total lines including empty/comment lines
            - code_loc: Lines of actual code
            - entity_count: Number of entities found
            - total_complexity: Sum of entity complexities
            - avg_complexity: Average complexity per entity
        """
        total_loc = parser_utils.count_lines(content)
        code_loc = parser_utils.count_lines_of_code(content)

        total_complexity = sum(e.complexity for e in entities)

        return {
            "total_loc": total_loc,
            "code_loc": code_loc,
            "entity_count": len(entities),
            "total_complexity": total_complexity,
            "avg_complexity": total_complexity / len(entities) if entities else 0,
        }

    DEFINITION_TYPES = {
        "function_definition",
        "class_definition",
        "method_definition",
        "function_declaration",
        "class_declaration",
    }

    CALL_TYPES = {
        "call_expression",
        "method_call",
        "identifier",
    }

    IMPORT_TYPES = {
        "import_statement",
        "import_from_statement",
        "require_call",
    }

    def __init__(self):
        super().__init__()
        self.parsers: Dict[str, Parser] = {}
        self._load_languages()

    def _load_languages(self):
        """Load tree-sitter language parsers."""
        if not TREE_SITTER_AVAILABLE:
            return

        try:
            from tree_sitter_languages import get_language, get_parser

            for lang, _ in self.LANGUAGE_CONFIG.items():
                self.parsers[lang] = get_parser(lang)
        except Exception:
            pass

    def is_available(self) -> bool:
        """Check if tree-sitter is available."""
        return TREE_SITTER_AVAILABLE and len(self.parsers) > 0

    def parse_file(self, file_path: Path) -> ParseResult:
        """Parse a file using tree-sitter."""
        content = self.read_file(file_path)
        if content is None:
            return ParseResult(
                file=str(file_path), language="unknown", errors=["Could not read file"]
            )
        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Parse file content using tree-sitter."""
        result = ParseResult(
            file=str(file_path),
            language=parser_utils.detect_language(file_path) or "unknown",
        )

        if not TREE_SITTER_AVAILABLE:
            result.errors.append("tree-sitter not available")
            return result

        parser = self.parsers.get(result.language)
        if parser is None:
            result.errors.append(f"No parser for language: {result.language}")
            return result

        try:
            tree = parser.parse(bytes(content, "utf-8"))
            result.entities = self._extract_entities(tree.root_node, content, file_path)
            result.calls = self._extract_calls(tree.root_node, content)
            result.dependencies = self._extract_dependencies(tree.root_node, content)
            result.metrics = self._compute_metrics(content, result.entities)
        except Exception as e:
            result.errors.append(str(e))

        return result

    def _extract_entities(
        self, node, content: str, file_path: Path
    ) -> List[TreeSitterEntity]:
        """Extract entity definitions from AST."""
        entities = []
        lines = content.split("\n")

        for child in node.children:
            if child.type in self.DEFINITION_TYPES:
                entity = self._node_to_entity(child, content, file_path)
                if entity:
                    entities.append(entity)

            entities.extend(self._extract_entities(child, content, file_path))

        return entities

    def _node_to_entity(
        self, node, content: str, file_path: Path
    ) -> Optional[TreeSitterEntity]:
        """Convert a tree-sitter node to an Entity."""
        name = self._extract_name(node, content)
        if not name:
            return None

        line = node.start_point[0] + 1
        end_line = node.end_point[0] + 1
        loc = end_line - line

        entity_type = self._map_node_type(node.type)

        entity_id = self.generate_entity_id(entity_type, name)

        docstring = self._extract_docstring_from_node(node, content)

        return TreeSitterEntity(
            id=entity_id,
            type=entity_type,
            name=name,
            file=str(file_path),
            line=line,
            end_line=end_line,
            loc=loc,
            docstring=docstring,
            node_type=node.type,
            is_definition=True,
        )

    def _extract_name(self, node, content: str) -> Optional[str]:
        """Extract entity name from node."""
        lines = content.split("\n")

        for child in node.children:
            if child.type == "identifier":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()
            if child.type == "name":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()

        return None

    def _map_node_type(self, node_type: str) -> str:
        """Map tree-sitter node types to entity types."""
        type_map = {
            "function_definition": "function",
            "function_declaration": "function",
            "class_definition": "class",
            "class_declaration": "class",
            "method_definition": "method",
            "method_declaration": "method",
        }
        return type_map.get(node_type, "function")

    def _extract_docstring_from_node(self, node, content: str) -> Optional[str]:
        """Extract docstring from node."""
        lines = content.split("\n")
        start_line = node.start_point[0]

        if start_line + 1 < len(lines):
            first_line = lines[start_line].strip()
            if first_line.startswith('"""') or first_line.startswith("'''"):
                end_char = first_line[3:]
                if end_char and not end_char.endswith("\\"):
                    return first_line[3:-3]

        return None

    def _extract_calls(self, node, content: str) -> List[CallEdge]:
        """Extract function calls from AST."""
        calls = []

        for child in node.children:
            if child.type in self.CALL_TYPES:
                call = self._node_to_call(child, content)
                if call:
                    calls.append(call)

            calls.extend(self._extract_calls(child, content))

        return calls

    def _node_to_call(self, node, content: str) -> Optional[CallEdge]:
        """Convert a call node to a CallEdge."""
        lines = content.split("\n")
        line = node.start_point[0] + 1

        if node.children:
            callee = node.children[0]
            if callee.type == "identifier":
                line_idx = callee.start_point[0]
                if line_idx < len(lines):
                    callee_name = lines[line_idx].strip()
                    return CallEdge(
                        from_id="",
                        to_id=f"func:{callee_name}",
                        line=line,
                    )

        return None

    def _extract_dependencies(self, node, content: str) -> List[Dependency]:
        """Extract module dependencies from AST."""
        deps = []

        for child in node.children:
            if child.type in self.IMPORT_TYPES:
                dep = self._node_to_dependency(child, content)
                if dep:
                    deps.append(dep)

            deps.extend(self._extract_dependencies(child, content))

        return deps

    def _node_to_dependency(self, node, content: str) -> Optional[Dependency]:
        """Convert import node to Dependency."""
        lines = content.split("\n")

        for child in node.children:
            if child.type == "string":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    module = lines[line_idx].strip().strip('"').strip("'")
                    return Dependency(
                        source_module="",
                        target_module=module,
                        dependency_type="import",
                    )

        return None

    def _compute_metrics(self, content: str, entities: List[Entity]) -> Dict[str, Any]:
        """Compute metrics for the parsed content."""
        total_loc = parser_utils.count_lines(content)
        code_loc = parser_utils.count_lines_of_code(content)

        total_complexity = sum(e.complexity for e in entities)

        return {
            "total_loc": total_loc,
            "code_loc": code_loc,
            "entity_count": len(entities),
            "total_complexity": total_complexity,
            "avg_complexity": total_complexity / len(entities) if entities else 0,
        }
