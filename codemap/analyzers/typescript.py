"""Analizador de TypeScript para CodeMap.

Analiza archivos fuente de TypeScript para extraer entidades, construir grafos de llamadas
y calcular métricas de calidad usando tree-sitter para un parseo preciso.

Ejemplo
-------
>>> from codemap.analyzers.typescript import TypeScriptAnalyzer
>>> analyzer = TypeScriptAnalyzer()
>>> result = analyzer.analyze_file(Path("example.ts"))
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


class TypeScriptASTParser(BaseASTParser):
    """Parser AST de TypeScript usando tree-sitter.

    Extrae clases, interfaces, funciones, métodos, importaciones y relaciones de llamadas.
    Maneja constructos específicos de TypeScript como interfaces, tipos y genéricos.
    """

    SUPPORTED_LANGUAGES = ["typescript"]

    def parse_file(self, file_path: Path) -> "ParseResult":
        """Parsea un archivo fuente de TypeScript.

        Args:
            file_path: Ruta del archivo TypeScript.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="typescript",
        )

        content = self.read_file(file_path)
        if content is None:
            result.errors.append(f"Could not read file: {file_path}")
            return result

        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> "ParseResult":
        """Parsea código fuente de TypeScript desde un string.

        Args:
            content: Código fuente de TypeScript.
            file_path: Ruta para referencia.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="typescript",
        )

        try:
            import tree_sitter

            from tree_sitter_languages import get_language

            language = get_language("typescript")
            parser = tree_sitter.Parser(language)

            tree = parser.parse(bytes(content, "utf-8"))

            # Extract entities
            entities = self._extract_entities(tree.root_node, str(file_path))
            result.entities = entities

            # Extract calls
            calls = self._extract_calls(tree.root_node, entities, content)
            result.calls = calls

            # Extract dependencies
            dependencies = self._extract_dependencies(tree.root_node)
            result.dependencies = dependencies

        except Exception as e:
            result.errors.append(f"Parse error: {str(e)}")

        return result

    def _extract_entities(self, node, file_str: str) -> List["Entity"]:
        """Extrae entidades de clases, interfaces y funciones del AST.

        Args:
            node: Nodo raíz del AST.
            file_str: Ruta del archivo como string.

        Returns:
            Lista de objetos Entity.
        """
        from codemap.parsers.ast_parser import Entity

        entities = []

        # Handle class declarations
        if node.type in ["class_declaration", "CLASS_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                methods = self._get_class_methods(node)
                line = self._get_line(node)
                end_line = self._get_end_line(node)

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

        # Handle interface declarations
        if node.type in ["interface_declaration", "INTERFACE_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                properties = self._get_interface_properties(node)
                line = self._get_line(node)

                entity = Entity(
                    id=self.generate_entity_id("interface", name),
                    type="interface",
                    name=name,
                    file=file_str,
                    line=line,
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

        # Handle function declarations
        if node.type in ["function_declaration", "FUNCTION_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                line = self._get_line(node)
                entity = Entity(
                    id=self.generate_entity_id("function", name),
                    type="function",
                    name=name,
                    file=file_str,
                    line=line,
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

        # Handle arrow functions
        if node.type in ["arrow_function", "ARROW_FUNCTION"]:
            line = self._get_line(node)
            name = f"arrow_{line}"
            entity = Entity(
                id=self.generate_entity_id("function", name),
                type="function",
                name=name,
                file=file_str,
                line=line,
                complexity=self._estimate_complexity(node),
            )
            entities.append(entity)

        # Recursively search children
        for child in node.children:
            entities.extend(self._extract_entities(child, file_str))

        return entities

    def _get_identifier_name(self, node) -> Optional[str]:
        """Get the name from an identifier node."""
        if node.type in ["identifier", "IDENTIFIER"]:
            text = node.text
            return text.decode("utf-8") if hasattr(text, "decode") else text

        for child in node.children:
            name = self._get_identifier_name(child)
            if name:
                return name
        return None

    def _get_class_methods(self, node) -> List[str]:
        """Get list of method names from class."""
        methods = []
        for child in node.children:
            if child.type in ["method_definition", "METHOD_DEFINITION", "method_signature"]:
                name = self._get_identifier_name(child)
                if name and not name.startswith("_"):
                    methods.append(name)
        return methods

    def _get_interface_properties(self, node) -> List[str]:
        """Get list of property names from interface."""
        properties = []
        for child in node.children:
            if child.type in ["property_signature", "PROPERTY_SIGNATURE"]:
                name = self._get_identifier_name(child)
                if name:
                    properties.append(name)
        return properties

    def _get_line(self, node) -> int:
        """Get the line number (1-indexed) from a node."""
        if hasattr(node, "start_point") and node.start_point:
            return node.start_point[0] + 1
        return 0

    def _get_end_line(self, node) -> int:
        """Get the end line number (1-indexed) from a node."""
        if hasattr(node, "end_point") and node.end_point:
            return node.end_point[0] + 1
        return self._get_line(node)

    def _extract_calls(
        self, node, entities: List["Entity"], content: str
    ) -> List["CallEdge"]:
        """Extrae las relaciones de llamadas del AST."""
        from codemap.parsers.ast_parser import CallEdge

        calls = []
        entity_names = {e.name for e in entities}

        for child in node.children:
            if child.type in ["call_expression", "CALL_EXPRESSION"]:
                called_name = self._get_called_name(child)
                if called_name in entity_names:
                    caller = self._find_caller_entity(child, entities)
                    if caller:
                        line = self._get_line(child)
                        calls.append(
                            CallEdge(
                                from_id=caller.id,
                                to_id=self.generate_entity_id("function", called_name),
                                call_type="direct_call",
                                line=line,
                            )
                        )

            calls.extend(self._extract_calls(child, entities, content))

        return calls

    def _get_called_name(self, node) -> Optional[str]:
        """Get the name of a called function."""
        for child in node.children:
            if child.type in ["identifier", "IDENTIFIER"]:
                text = child.text
                return text.decode("utf-8") if hasattr(text, "decode") else text
        return None

    def _find_caller_entity(self, node, entities: List["Entity"]) -> Optional["Entity"]:
        """Find which entity contains a call."""
        for entity in entities:
            if entity.line:
                call_line = self._get_line(node)
                if entity.line <= call_line <= (entity.end_line or entity.line):
                    return entity
        return None

    def _extract_dependencies(self, node) -> List["Dependency"]:
        """Extrae declaraciones de import."""
        from codemap.parsers.ast_parser import Dependency

        dependencies = []

        for child in node.children:
            if child.type in ["import_statement", "IMPORT_STATEMENT"]:
                module = self._get_import_module(child)
                if module:
                    dependencies.append(
                        Dependency(
                            source_module="",
                            target_module=module,
                            dependency_type="import",
                        )
                    )

            dependencies.extend(self._extract_dependencies(child))

        return dependencies

    def _get_import_module(self, node) -> Optional[str]:
        """Get module name from import statement."""
        for child in node.children:
            if child.type in ["string", "STRING", "string_fragment", "STRING_FRAGMENT"]:
                text = child.text.decode("utf-8").strip('"\'') if hasattr(child.text, "decode") else child.text.strip('"\'')
                if text and not text.startswith("."):
                    return text
        return None

    def _estimate_complexity(self, node) -> int:
        """Estimate cyclomatic complexity."""
        complexity = 1

        for child in node.children:
            child_type = child.type
            if child_type in ["if_statement", "IF_STATEMENT"]:
                complexity += 1
            elif child_type in ["for_statement", "FOR_STATEMENT"]:
                complexity += 1
            elif child_type in ["while_statement", "WHILE_STATEMENT"]:
                complexity += 1
            elif child_type in ["catch_clause", "CATCH_CLAUSE"]:
                complexity += 1
            elif child_type in ["ternary_expression", "CONDITIONAL_EXPRESSION"]:
                complexity += 1

            complexity += self._estimate_complexity(child) - 1

        return complexity


class TypeScriptAnalyzer(BaseAnalyzer):
    """Analizador del lenguaje TypeScript.

    Maneja archivos .ts, .tsx, .mts, .cts usando tree-sitter para parsear.
    Soporta características específicas de TypeScript como interfaces, tipos y genéricos.

    Atributos:
        SUPPORTED_EXTENSIONS: Extensiones de archivos que maneja este analizador
        SUPPORTED_LANGUAGES: Identificadores de lenguajes que soporta este analizador
    """

    SUPPORTED_EXTENSIONS = [".ts", ".tsx", ".mts", ".cts"]
    SUPPORTED_LANGUAGES = ["typescript"]

    def _create_parser(self) -> BaseASTParser:
        """Crea un parser AST de TypeScript.

        Returns:
            Instancia de TypeScriptASTParser.
        """
        return TypeScriptASTParser()

    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si este analizador puede manejar el archivo TypeScript dado.

        Args:
            file_path: Ruta del archivo.

        Returns:
            True si es un archivo de TypeScript.
        """
        return parser_utils.detect_language(file_path) == "typescript"

    def _extract_metrics(self, content: str, entities: List) -> Dict[str, Any]:
        """Calcula métricas de calidad específicas de TypeScript.

        Args:
            content: Código fuente de TypeScript.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario de métricas calculadas.
        """
        metrics = {}

        # Basic counts
        lines = content.split("\n")
        metrics["total_lines"] = len(lines)
        metrics["code_lines"] = parser_utils.count_lines_of_code(content)

        # Comment lines
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
        interfaces = [e for e in entities if e.type == "interface"]
        functions = [e for e in entities if e.type == "function"]

        metrics["class_count"] = len(classes)
        metrics["interface_count"] = len(interfaces)
        metrics["function_count"] = len(functions)

        # Complexity
        if entities:
            max_complexity = max(e.complexity for e in entities)
            avg_complexity = sum(e.complexity for e in entities) / len(entities)
            metrics["max_complexity"] = max_complexity
            metrics["avg_complexity"] = round(avg_complexity, 2)

        # TypeScript-specific features
        metrics["uses_generic"] = "<" in content and ">" in content
        metrics["uses_type_alias"] = "type " in content
        metrics["uses_interface"] = "interface " in content
        metrics["uses_enum"] = "enum " in content
        metrics["uses_decorator"] = "@" in content

        # Async/await
        metrics["uses_async"] = "async " in content or "await " in content

        # Export types
        metrics["has_default_export"] = "export default" in content
        metrics["has_type_export"] = "export type" in content

        # Generics usage count
        generic_count = content.count("<")
        metrics["generic_usage_count"] = generic_count

        # Type annotation density (rough estimate)
        colon_count = content.count(":")
        if metrics["code_lines"] > 0:
            metrics["type_annotation_density"] = round(colon_count / metrics["code_lines"], 2)

        # Strict mode indicators
        metrics["has_strict_flag"] = '"strict"' in content or "'strict'" in content

        return metrics
