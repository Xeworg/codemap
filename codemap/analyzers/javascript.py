"""Analizador de JavaScript para CodeMap.

Analiza archivos fuente de JavaScript para extraer entidades, construir grafos de llamadas
y calcular métricas de calidad usando tree-sitter para un parseo preciso.

Ejemplo
-------
>>> from codemap.analyzers.javascript import JavaScriptAnalyzer
>>> analyzer = JavaScriptAnalyzer()
>>> result = analyzer.analyze_file(Path("example.js"))
>>> len(result.entities)
5
"""

from __future__ import annotations

from pathlib import Path
from typing import List, Dict, Any, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

from codemap.parsers.ast_parser import BaseASTParser
from codemap.analyzers.base import BaseAnalyzer, AnalysisResult
from codemap.parsers import utils as parser_utils


class JavaScriptASTParser(BaseASTParser):
    """Parser AST de JavaScript usando tree-sitter.

    Extrae clases, funciones, métodos, importaciones y relaciones de llamadas.
    """

    SUPPORTED_LANGUAGES = ["javascript"]

    def __init__(self):
        super().__init__()
        self._content = ""

    def _create_node_id(self, node_type: str, name: str, line: int) -> str:
        """Generate a unique node ID."""
        return f"{node_type}:{name}@{line}"

    def parse_file(self, file_path: Path) -> "ParseResult":
        """Parsea un archivo fuente de JavaScript.

        Args:
            file_path: Ruta del archivo JavaScript.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="javascript",
        )

        content = self.read_file(file_path)
        if content is None:
            result.errors.append(f"Could not read file: {file_path}")
            return result

        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> "ParseResult":
        """Parsea código fuente de JavaScript desde un string.

        Args:
            content: Código fuente de JavaScript.
            file_path: Ruta para referencia.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="javascript",
        )

        self._content = content

        try:
            import tree_sitter

            # Load JavaScript language
            from tree_sitter_languages import get_language

            language = get_language("javascript")
            parser = tree_sitter.Parser(language)

            tree = parser.parse(bytes(content, "utf-8"))

            # Extract entities
            entities = self._extract_entities(tree.root_node, str(file_path))
            result.entities = entities

            # Extract calls
            calls = self._extract_calls(tree.root_node, entities, content)
            result.calls = calls

            # Extract dependencies (imports/requires)
            dependencies = self._extract_dependencies(tree.root_node)
            result.dependencies = dependencies

        except Exception as e:
            result.errors.append(f"Parse error: {str(e)}")

        return result

    def _extract_entities(self, node, file_str: str) -> List[Entity]:
        """Extrae entidades de clases y funciones del AST.

        Args:
            node: Nodo raíz del AST.
            file_str: Ruta del archivo como string.

        Returns:
            Lista de objetos Entity.
        """
        entities = []

        if node.type in ["class_declaration", "CLASS"]:
            name = self._get_class_name(node)
            if name:
                methods = self._get_class_methods(node)
                line = node.start_point[0] + 1 if node.start_point else 0
                end_line = node.end_point[0] + 1 if node.end_point else line

                entity = Entity(
                    id=self.generate_entity_id("class", name),
                    type="class",
                    name=name,
                    file=file_str,
                    line=line,
                    end_line=end_line,
                    methods=methods,
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

                # Recursively find methods
                for child in node.children:
                    if child.type in ["method_definition", "METHOD"]:
                        method_entity = self._extract_method(child, file_str, entity.id)
                        if method_entity:
                            entities.append(method_entity)

        # Look for functions and arrow functions
        for child in node.children:
            if child.type in ["function_declaration", "FUNCTION"]:
                entity = self._extract_function(child, file_str, None)
                if entity:
                    entities.append(entity)
            elif child.type in ["arrow_function", "ARROW"]:
                entity = self._extract_arrow_function(child, file_str, None)
                if entity:
                    entities.append(entity)

            # Recursively search
            if child.child_count > 0:
                entities.extend(self._extract_entities(child, file_str))

        return entities

    def _get_class_name(self, node) -> Optional[str]:
        """Get class name from class declaration."""
        for child in node.children:
            if child.type in ["identifier", "NAME"]:
                return child.text.decode("utf-8") if hasattr(child.text, "decode") else child.text
        return None

    def _get_class_methods(self, node) -> List[str]:
        """Get list of method names from class."""
        methods = []
        for child in node.children:
            if child.type in ["method_definition", "METHOD"]:
                for subchild in child.children:
                    if subchild.type in ["identifier", "property_name"]:
                        name = subchild.text.decode("utf-8") if hasattr(subchild.text, "decode") else subchild.text
                        if name and not name.startswith("_"):
                            methods.append(name)
        return methods

    def _extract_method(self, node, file_str: str, parent_id: Optional[str]) -> Optional[Entity]:
        """Extract a method entity."""
        name = None
        for child in node.children:
            if child.type in ["identifier", "property_name"]:
                name = child.text.decode("utf-8") if hasattr(child.text, "decode") else child.text
                break

        if not name:
            return None

        line = node.start_point[0] + 1 if node.start_point else 0

        return Entity(
            id=self.generate_entity_id("method", name),
            type="method",
            name=name,
            file=file_str,
            line=line,
            parent=parent_id,
            complexity=self._estimate_complexity(node),
        )

    def _extract_function(self, node, file_str: str, parent_id: Optional[str]) -> Optional[Entity]:
        """Extract a function declaration."""
        name = None
        for child in node.children:
            if child.type == "identifier":
                name = child.text.decode("utf-8") if hasattr(child.text, "decode") else child.text
                break

        if not name:
            return None

        line = node.start_point[0] + 1 if node.start_point else 0

        return Entity(
            id=self.generate_entity_id("function", name),
            type="function",
            name=name,
            file=file_str,
            line=line,
            parent=parent_id,
            complexity=self._estimate_complexity(node),
        )

    def _extract_arrow_function(self, node, file_str: str, parent_id: Optional[str]) -> Optional[Entity]:
        """Extract an arrow function (anonymous)."""
        # Arrow functions are anonymous, generate a synthetic name
        line = node.start_point[0] + 1 if node.start_point else 0
        name = f"arrow_{line}"

        return Entity(
            id=self.generate_entity_id("function", name),
            type="function",
            name=name,
            file=file_str,
            line=line,
            parent=parent_id,
            complexity=self._estimate_complexity(node),
        )

    def _extract_calls(
        self, node, entities: List[Entity], content: str
    ) -> List[CallEdge]:
        """Extrae las relaciones de llamadas del AST."""
        calls = []
        entity_names = {e.name for e in entities}

        for child in node.children:
            if child.type in ["call_expression", "CALL"]:
                called_name = self._get_called_name(child)
                if called_name in entity_names:
                    caller = self._find_caller_entity(child, entities)
                    if caller:
                        line = child.start_point[0] + 1 if child.start_point else 0
                        calls.append(
                            CallEdge(
                                from_id=caller.id,
                                to_id=self.generate_entity_id("function", called_name),
                                call_type="direct_call",
                                line=line,
                            )
                        )

            if child.child_count > 0:
                calls.extend(self._extract_calls(child, entities, content))

        return calls

    def _get_called_name(self, node) -> Optional[str]:
        """Get the name of a called function."""
        for child in node.children:
            if child.type in ["identifier", "NAME"]:
                return child.text.decode("utf-8") if hasattr(child.text, "decode") else child.text
            # Handle object.method calls
            if child.type in ["member_expression", "MEMBER"]:
                parts = []
                self._get_member_parts(child, parts)
                return parts[-1] if parts else None
        return None

    def _get_member_parts(self, node, parts: List[str]):
        """Recursively get parts of member expression."""
        for child in node.children:
            if child.type in ["identifier", "NAME"]:
                name = child.text.decode("utf-8") if hasattr(child.text, "decode") else child.text
                parts.append(name)
            elif child.child_count > 0:
                self._get_member_parts(child, parts)

    def _find_caller_entity(self, node, entities: List[Entity]) -> Optional[Entity]:
        """Find which entity contains a call."""
        for entity in entities:
            if entity.line and node.start_point:
                call_line = node.start_point[0] + 1
                if entity.line <= call_line <= (entity.end_line or entity.line):
                    return entity
        return None

    def _extract_dependencies(self, node) -> List[Dependency]:
        """Extrae declaraciones de import y require."""
        dependencies = []

        for child in node.children:
            if child.type in ["import_statement", "IMPORT"]:
                modules = self._get_import_modules(child)
                for module in modules:
                    dependencies.append(
                        Dependency(
                            source_module="",
                            target_module=module,
                            dependency_type="import",
                        )
                    )
            elif child.type in ["require", "REQUIRE"]:
                module = self._get_require_module(child)
                if module:
                    dependencies.append(
                        Dependency(
                            source_module="",
                            target_module=module,
                            dependency_type="require",
                        )
                    )

            if child.child_count > 0:
                dependencies.extend(self._extract_dependencies(child))

        return dependencies

    def _get_import_modules(self, node) -> List[str]:
        """Get module names from import statement."""
        modules = []
        for child in node.children:
            if child.type in ["string", "STRING"]:
                module = child.text.decode("utf-8").strip('"\'')
                if module and not module.startswith("."):
                    modules.append(module)
        return modules

    def _get_require_module(self, node) -> Optional[str]:
        """Get module name from require call."""
        for child in node.children:
            if child.type in ["string", "STRING"]:
                return child.text.decode("utf-8").strip('"\'')
        return None

    def _estimate_complexity(self, node) -> int:
        """Estimate cyclomatic complexity."""
        complexity = 1

        for child in node.children:
            child_type = child.type
            if child_type in ["if_statement", "IF", "else_clause", "ELSE"]:
                complexity += 1
            elif child_type in ["for_statement", "FOR", "for_in_statement"]:
                complexity += 1
            elif child_type in ["while_statement", "WHILE"]:
                complexity += 1
            elif child_type in ["catch_clause", "CATCH"]:
                complexity += 1
            elif child_type in ["ternary_expression", "CONDITIONAL"]:
                complexity += 1

            if child.child_count > 0:
                complexity += self._estimate_complexity(child) - 1

        return complexity


class JavaScriptAnalyzer(BaseAnalyzer):
    """Analizador del lenguaje JavaScript.

    Maneja archivos .js, .jsx, .mjs, .cjs usando tree-sitter para parsear.

    Atributos:
        SUPPORTED_EXTENSIONS: Extensiones de archivos que maneja este analizador
        SUPPORTED_LANGUAGES: Identificadores de lenguajes que soporta este analizador
    """

    SUPPORTED_EXTENSIONS = [".js", ".jsx", ".mjs", ".cjs"]
    SUPPORTED_LANGUAGES = ["javascript"]

    def _create_parser(self) -> BaseASTParser:
        """Crea un parser AST de JavaScript.

        Returns:
            Instancia de JavaScriptASTParser.
        """
        return JavaScriptASTParser()

    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si este analizador puede manejar el archivo JavaScript dado.

        Args:
            file_path: Ruta del archivo.

        Returns:
            True si es un archivo de JavaScript.
        """
        return parser_utils.detect_language(file_path) == "javascript"

    def _extract_metrics(self, content: str, entities: List) -> Dict[str, Any]:
        """Calcula métricas de calidad específicas de JavaScript.

        Args:
            content: Código fuente de JavaScript.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario de métricas calculadas.
        """
        metrics = {}

        # Basic counts
        lines = content.split("\n")
        metrics["total_lines"] = len(lines)

        # Code lines
        metrics["code_lines"] = parser_utils.count_lines_of_code(content)

        # Comment lines (// and /* */)
        comment_lines = 0
        in_multiline = False
        for line in lines:
            stripped = line.strip()
            if in_multiline:
                comment_lines += 1
                if "*/" in stripped:
                    in_multiline = False
            elif stripped.startswith("//"):
                comment_lines += 1
            elif stripped.startswith("/*"):
                comment_lines += 1
                if "*/" not in stripped:
                    in_multiline = True

        metrics["comment_lines"] = comment_lines

        # Entity counts
        classes = [e for e in entities if e.type == "class"]
        functions = [e for e in entities if e.type == "function"]
        methods = [e for e in entities if e.type == "method"]

        metrics["class_count"] = len(classes)
        metrics["function_count"] = len(functions)
        metrics["method_count"] = len(methods)

        # Complexity
        if entities:
            max_complexity = max(e.complexity for e in entities)
            avg_complexity = sum(e.complexity for e in entities) / len(entities)
            metrics["max_complexity"] = max_complexity
            metrics["avg_complexity"] = round(avg_complexity, 2)

        # Arrow function ratio
        arrow_count = len([f for f in functions if f.name.startswith("arrow_")])
        if functions:
            metrics["arrow_function_ratio"] = round(arrow_count / len(functions), 2)

        # ES6+ features detection
        metrics["uses_await"] = "await " in content or " async " in content
        metrics["uses_promises"] = ".then(" in content or "Promise" in content
        metrics["uses_destructuring"] = "const {" in content or "let {" in content

        # Export detection
        metrics["has_default_export"] = "export default" in content
        metrics["has_named_exports"] = "export " in content and "export default" not in content

        # Import count
        import_count = content.count("import ")
        require_count = content.count("require(")
        metrics["import_count"] = import_count
        metrics["require_count"] = require_count

        return metrics
