"""Python AST Parser for CodeMap.

Parses Python source files using libcst for accurate extraction of
classes, functions, methods, imports, and call relationships.

Example
-------
>>> from codemap.parsers.python_parser import PythonASTParser
>>> parser = PythonASTParser()
>>> result = parser.parse_file(Path("example.py"))
>>> len(result.entities)
5
"""

from __future__ import annotations

import ast
from pathlib import Path
from typing import List, Optional, Dict, Any

from codemap.parsers.ast_parser import (
    BaseASTParser,
    ParseResult,
    Entity,
    CallEdge,
    Dependency,
)


def _get_name_value(node: Any) -> str:
    """Extract string value from a Name or Attribute node."""
    if node is None:
        return ""
    if hasattr(node, 'value'):
        return node.value
    if hasattr(node, 'name'):
        return node.name
    return str(node)


def _get_module_name(node: Any) -> str:
    """Extract module name from an ImportFrom statement."""
    if node is None:
        return ""
    parts = []
    current = node
    while current is not None:
        if hasattr(current, 'attr') and current.attr:
            parts.append(current.attr.value)
            current = current.value
        elif hasattr(current, 'name') and current.name:
            parts.append(current.name.value)
            break
        else:
            break
    return ".".join(reversed(parts)) if parts else str(node)


def _get_line(node: Any) -> int:
    """Get line number from a libcst node."""
    if hasattr(node, 'start') and node.start:
        return node.start.line
    return 0


def _get_end_line(node: Any) -> int:
    """Get end line number from a libcst node."""
    if hasattr(node, 'end') and node.end:
        return node.end.line
    return _get_line(node)


class PythonASTParser(BaseASTParser):
    """Python-specific AST parser using libcst for accurate parsing.

    Extracts:
    - Classes with their methods
    - Functions (top-level and nested)
    - Import statements
    - Function calls within the code
    """

    SUPPORTED_LANGUAGES = ["python"]

    def __init__(self):
        """Initialize the Python parser."""
        super().__init__()
        self.imports: List[str] = []
        self.imported_symbols: Dict[str, set] = {}

    def parse_file(self, file_path: Path) -> ParseResult:
        """Parse a Python source file."""
        result = ParseResult(
            file=str(file_path),
            language="python",
        )

        content = self.read_file(file_path)
        if content is None:
            result.errors.append(f"Could not read file: {file_path}")
            return result

        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Parse Python source code from a string."""
        result = ParseResult(
            file=str(file_path),
            language="python",
        )

        try:
            import libcst as cst

            tree = cst.parse_module(content)

            # Reset state
            self.imports = []
            self.imported_symbols = {}

            # Extract imports first
            self._extract_imports(tree, cst)

            # Extract entities from the tree
            entities = self._extract_entities(tree, file_path, cst)
            result.entities = entities

            # Build call edges
            calls = self._extract_calls(content, entities)
            result.calls = calls

            # Build dependencies
            dependencies = self._build_dependencies()
            result.dependencies = dependencies

            # Extract docstrings
            self._extract_docstrings(tree, entities, content)

        except SyntaxError as e:
            result.errors.append(f"Syntax error: {str(e)}")
        except Exception as e:
            result.errors.append(f"Parse error: {str(e)}")

        return result

    def _extract_imports(self, tree: Any, cst: Any) -> None:
        """Extract all import statements from the AST."""
        for node in tree.body:
            if isinstance(node, cst.SimpleStatementLine):
                stmt = node.body[0]
                if isinstance(stmt, cst.Import):
                    for alias in stmt.names:
                        name = alias.name.value
                        self.imports.append(name)
                        self.imported_symbols[name] = {name}

                elif isinstance(stmt, cst.ImportFrom):
                    module = stmt.module
                    module_name = _get_module_name(module) if module else ""
                    if module_name:
                        self.imports.append(module_name)
                        symbols = set()
                        for alias in stmt.names:
                            if alias.asname is not None:
                                symbols.add(alias.asname.name.value)
                            else:
                                symbols.add(alias.name.value)
                        self.imported_symbols[module_name] = symbols

    def _extract_entities(self, tree: Any, file_path: Path, cst: Any) -> List[Entity]:
        """Extract classes and functions from the AST."""
        entities = []
        file_str = str(file_path)

        for node in tree.body:
            if isinstance(node, cst.ClassDef):
                entity = self._create_class_entity(node, file_str, cst)
                entities.append(entity)

            elif isinstance(node, cst.FunctionDef):
                entity = self._create_function_entity(node, file_str, None, cst)
                entities.append(entity)

        return entities

    def _create_class_entity(self, node: Any, file_str: str, cst: Any) -> Entity:
        """Create an Entity for a class definition."""
        name = node.name.value

        # Extract methods
        methods = []
        for item in node.body.body:
            if isinstance(item, cst.FunctionDef):
                methods.append(item.name.value)

        # Get line numbers
        line = _get_line(node)
        end_line = _get_end_line(node)

        # Count lines of code
        loc = (end_line or line) - line + 1 if line else 0

        entity_id = self.generate_entity_id("class", name)

        return Entity(
            id=entity_id,
            type="class",
            name=name,
            file=file_str,
            line=line,
            end_line=end_line,
            methods=methods,
            docstring="",
            loc=loc,
            complexity=self._estimate_complexity(node, cst),
        )

    def _create_function_entity(
        self,
        node: Any,
        file_str: str,
        parent_class: Optional[str],
        cst: Any,
    ) -> Entity:
        """Create an Entity for a function definition."""
        name = node.name.value
        is_method = parent_class is not None
        entity_type = "method" if is_method else "function"

        # Extract parameters
        params = []
        if node.params.params:
            for param in node.params.params:
                params.append(param.name.value)

        line = _get_line(node)
        end_line = _get_end_line(node)

        loc = (end_line or line) - line + 1 if line else 0

        entity_id = self.generate_entity_id(entity_type, name)

        return Entity(
            id=entity_id,
            type=entity_type,
            name=name,
            file=file_str,
            line=line,
            end_line=end_line,
            parent=parent_class,
            parameters=params,
            docstring="",
            loc=loc,
            complexity=self._estimate_complexity(node, cst),
        )

    def _extract_docstrings(
        self, tree: Any, entities: List[Entity], content: str
    ) -> None:
        """Extract docstrings for all entities."""
        lines = content.split("\n")

        for entity in entities:
            docstring = self._extract_docstring_from_line(
                lines, entity.line
            )
            if docstring:
                entity.docstring = docstring

    def _extract_docstring_from_line(
        self, lines: List[str], start_line: int
    ) -> Optional[str]:
        """Extract docstring starting from a specific line."""
        if start_line - 1 >= len(lines):
            return None

        line = lines[start_line - 1].strip()

        if line.startswith('"""') or line.startswith("'''"):
            quote = line[:3]
            inner = line[3:]

            # Single line docstring
            closing = inner.find(quote)
            if closing != -1:
                return inner[:closing].strip()

            # Multi-line docstring
            for i in range(start_line, len(lines)):
                if quote in lines[i]:
                    doc_lines = lines[start_line : i + 1]
                    full_doc = "\n".join(doc_lines)
                    return full_doc.strip().strip(quote).strip()

        return None

    def _extract_calls(
        self, content: str, entities: List[Entity]
    ) -> List[CallEdge]:
        """Extract function calls between entities."""
        calls = []
        entity_names = {e.name for e in entities}
        method_map = {e.name: e.id for e in entities if e.type == "method"}

        try:
            tree = ast.parse(content)

            for node in ast.walk(tree):
                if isinstance(node, ast.Call):
                    call_name = self._get_call_name(node)

                    if call_name in entity_names and call_name in method_map:
                        caller = self._get_caller_entity(node, entities)
                        if caller:
                            calls.append(
                                CallEdge(
                                    from_id=caller.id,
                                    to_id=method_map[call_name],
                                    call_type="direct_call",
                                    line=node.lineno,
                                )
                            )

        except SyntaxError:
            pass

        return calls

    def _get_call_name(self, node: ast.Call) -> str:
        """Get the name of a function call."""
        if isinstance(node.func, ast.Name):
            return node.func.id
        elif isinstance(node.func, ast.Attribute):
            return node.func.attr
        return ""

    def _get_caller_entity(
        self, node: ast.Call, entities: List[Entity]
    ) -> Optional[Entity]:
        """Find which entity contains a call."""
        for entity in entities:
            if entity.line and node.lineno:
                if entity.line <= node.lineno <= (entity.end_line or entity.line):
                    return entity
        return None

    def _build_dependencies(self) -> List[Dependency]:
        """Build module dependency list from extracted imports."""
        dependencies = []

        for module in self.imports:
            dependency = Dependency(
                source_module="",
                target_module=module,
                symbols=list(self.imported_symbols.get(module, set())),
                dependency_type="import",
            )
            dependencies.append(dependency)

        return dependencies

    def _estimate_complexity(self, node: Any, cst: Any) -> int:
        """Estimate cyclomatic complexity of a node."""
        complexity = 1

        # Use depth-first walk to count decision points
        stack = [node]
        while stack:
            current = stack.pop()
            if current is None:
                continue

            # Check node type
            if isinstance(current, cst.If):
                complexity += 1
            elif isinstance(current, cst.For):
                complexity += 1
            elif isinstance(current, cst.While):
                complexity += 1
            elif isinstance(current, cst.Try):
                complexity += 1  # try
                for handler in current.handlers:
                    complexity += 1  # each except clause
            elif isinstance(current, cst.Assert):
                complexity += 1

            # Add children to stack
            if hasattr(current, '__iter__') and not isinstance(current, str):
                try:
                    for child in current.children:
                        stack.append(child)
                except (AttributeError, TypeError):
                    pass

        return complexity
