"""Analizador de Java para CodeMap.

Analiza archivos fuente de Java para extraer entidades, construir grafos de llamadas
y calcular métricas de calidad usando tree-sitter para un parseo preciso.

Ejemplo
-------
>>> from codemap.analyzers.java import JavaAnalyzer
>>> analyzer = JavaAnalyzer()
>>> result = analyzer.analyze_file(Path("Example.java"))
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


class JavaASTParser(BaseASTParser):
    """Parser AST de Java usando tree-sitter.

    Extrae clases, interfaces, enums, métodos, importaciones y relaciones de llamadas.
    Maneja constructos específicos de Java como anotaciones, genéricos y manejo de excepciones.
    """

    SUPPORTED_LANGUAGES = ["java"]

    def parse_file(self, file_path: Path) -> "ParseResult":
        """Parsea un archivo fuente de Java.

        Args:
            file_path: Ruta del archivo Java.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="java",
        )

        content = self.read_file(file_path)
        if content is None:
            result.errors.append(f"Could not read file: {file_path}")
            return result

        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> "ParseResult":
        """Parsea código fuente de Java desde un string.

        Args:
            content: Código fuente de Java.
            file_path: Ruta para referencia.

        Returns:
            ParseResult conteniendo las entidades y relaciones extraídas.
        """
        from codemap.parsers.ast_parser import ParseResult, Entity, CallEdge, Dependency

        result = ParseResult(
            file=str(file_path),
            language="java",
        )

        try:
            import tree_sitter

            from tree_sitter_languages import get_language

            language = get_language("java")
            parser = tree_sitter.Parser(language)

            tree = parser.parse(bytes(content, "utf-8"))

            # Extract entities
            entities = self._extract_entities(tree.root_node, str(file_path))
            result.entities = entities

            # Extract calls
            calls = self._extract_calls(tree.root_node, entities, content)
            result.calls = calls

            # Extract dependencies (imports)
            dependencies = self._extract_dependencies(tree.root_node)
            result.dependencies = dependencies

        except Exception as e:
            result.errors.append(f"Parse error: {str(e)}")

        return result

    def _extract_entities(self, node, file_str: str) -> List["Entity"]:
        """Extrae entidades de clases, interfaces, enums y métodos.

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
                parent = self._get_super_class(node)

                entity = Entity(
                    id=self.generate_entity_id("class", name),
                    type="class",
                    name=name,
                    file=file_str,
                    line=line,
                    end_line=end_line,
                    parent=parent,
                    methods=methods,
                    decorators=self._get_annotations(node),
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

        # Handle interface declarations
        if node.type in ["interface_declaration", "INTERFACE_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                methods = self._get_interface_methods(node)
                line = self._get_line(node)

                entity = Entity(
                    id=self.generate_entity_id("interface", name),
                    type="interface",
                    name=name,
                    file=file_str,
                    line=line,
                    methods=methods,
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

        # Handle enum declarations
        if node.type in ["enum_declaration", "ENUM_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                methods = self._get_enum_methods(node)
                line = self._get_line(node)

                entity = Entity(
                    id=self.generate_entity_id("enum", name),
                    type="enum",
                    name=name,
                    file=file_str,
                    line=line,
                    methods=methods,
                    complexity=self._estimate_complexity(node),
                )
                entities.append(entity)

        # Handle method declarations
        if node.type in ["method_declaration", "METHOD_DECLARATION"]:
            name = self._get_identifier_name(node)
            if name:
                line = self._get_line(node)
                params = self._get_method_parameters(node)

                entity = Entity(
                    id=self.generate_entity_id("method", name),
                    type="method",
                    name=name,
                    file=file_str,
                    line=line,
                    parameters=params,
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
            if child.type in ["method_declaration", "METHOD_DECLARATION"]:
                name = self._get_identifier_name(child)
                if name:
                    methods.append(name)
        return methods

    def _get_interface_methods(self, node) -> List[str]:
        """Get list of method signatures from interface."""
        methods = []
        for child in node.children:
            if child.type in ["method_declaration", "METHOD_DECLARATION", "method_signature"]:
                name = self._get_identifier_name(child)
                if name:
                    methods.append(name)
        return methods

    def _get_enum_methods(self, node) -> List[str]:
        """Get list of method names from enum."""
        methods = []
        for child in node.children:
            if child.type in ["method_declaration", "METHOD_DECLARATION"]:
                name = self._get_identifier_name(child)
                if name:
                    methods.append(name)
        return methods

    def _get_method_parameters(self, node) -> List[str]:
        """Get parameter names from method declaration."""
        params = []
        for child in node.children:
            if child.type in ["formal_parameter", "FORMAL_PARAMETER"]:
                name = self._get_identifier_name(child)
                if name:
                    params.append(name)
        return params

    def _get_super_class(self, node) -> Optional[str]:
        """Get superclass name from class declaration."""
        for child in node.children:
            if child.type in ["superclass", "SUPERCLASS"]:
                return self._get_identifier_name(child)
        return None

    def _get_annotations(self, node) -> List[str]:
        """Get list of annotations from a declaration."""
        annotations = []
        for child in node.children:
            if child.type in ["annotation", "ANNOTATION"]:
                name = self._get_annotation_name(child)
                if name:
                    annotations.append(name)
        return annotations

    def _get_annotation_name(self, node) -> Optional[str]:
        """Get annotation name."""
        for child in node.children:
            if child.type in ["identifier", "IDENTIFIER"]:
                text = child.text
                return text.decode("utf-8") if hasattr(text, "decode") else text
        return None

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

        for child in node.children:
            if child.type in ["method_invocation", "METHOD_INVOCATION"]:
                called_name = self._get_method_name(child)
                if called_name:
                    caller = self._find_caller_entity(child, entities)
                    if caller:
                        line = self._get_line(child)
                        calls.append(
                            CallEdge(
                                from_id=caller.id,
                                to_id=self.generate_entity_id("method", called_name),
                                call_type="direct_call",
                                line=line,
                            )
                        )

            calls.extend(self._extract_calls(child, entities, content))

        return calls

    def _get_method_name(self, node) -> Optional[str]:
        """Get the name of an invoked method."""
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
            if child.type in ["import_declaration", "IMPORT_DECLARATION"]:
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
        """Get fully qualified module name from import statement."""
        parts = []
        for child in node.children:
            if child.type in ["identifier", "IDENTIFIER"]:
                text = child.text
                name = text.decode("utf-8") if hasattr(text, "decode") else text
                parts.append(name)
            elif child.type in ["separator", "DOT"]:
                parts.append(".")

        if parts:
            return "".join(parts)
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


class JavaAnalyzer(BaseAnalyzer):
    """Analizador del lenguaje Java.

    Maneja archivos .java usando tree-sitter para parsear.
    Soporta características específicas de Java como interfaces, enums, anotaciones y genéricos.

    Atributos:
        SUPPORTED_EXTENSIONS: Extensiones de archivos que maneja este analizador
        SUPPORTED_LANGUAGES: Identificadores de lenguajes que soporta este analizador
    """

    SUPPORTED_EXTENSIONS = [".java"]
    SUPPORTED_LANGUAGES = ["java"]

    def _create_parser(self) -> BaseASTParser:
        """Crea un parser AST de Java.

        Returns:
            Instancia de JavaASTParser.
        """
        return JavaASTParser()

    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si este analizador puede manejar el archivo Java dado.

        Args:
            file_path: Ruta del archivo.

        Returns:
            True si es un archivo de Java.
        """
        return parser_utils.detect_language(file_path) == "java"

    def _extract_metrics(self, content: str, entities: List) -> Dict[str, Any]:
        """Calcula métricas de calidad específicas de Java.

        Args:
            content: Código fuente de Java.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario de métricas calculadas.
        """
        metrics = {}

        # Basic counts
        lines = content.split("\n")
        metrics["total_lines"] = len(lines)
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
        interfaces = [e for e in entities if e.type == "interface"]
        enums = [e for e in entities if e.type == "enum"]
        methods = [e for e in entities if e.type == "method"]

        metrics["class_count"] = len(classes)
        metrics["interface_count"] = len(interfaces)
        metrics["enum_count"] = len(enums)
        metrics["method_count"] = len(methods)

        # Complexity
        if entities:
            max_complexity = max(e.complexity for e in entities)
            avg_complexity = sum(e.complexity for e in entities) / len(entities)
            metrics["max_complexity"] = max_complexity
            metrics["avg_complexity"] = round(avg_complexity, 2)

        # Java-specific features
        metrics["uses_generic"] = "<" in content and ">" in content
        metrics["uses_annotation"] = "@" in content
        metrics["uses_interface"] = len(interfaces) > 0
        metrics["uses_enum"] = len(enums) > 0

        # Access modifiers (heuristic)
        public_count = content.count("public ")
        private_count = content.count("private ")
        protected_count = content.count("protected ")

        metrics["public_methods"] = public_count
        metrics["private_methods"] = private_count
        metrics["protected_methods"] = protected_count

        # Exception handling
        try_count = content.count("try {")
        catch_count = content.count("catch (")
        finally_count = content.count("finally {")

        metrics["try_blocks"] = try_count
        metrics["catch_blocks"] = catch_count
        metrics["finally_blocks"] = finally_count

        # Import count
        import_count = content.count("import ")
        metrics["import_count"] = import_count

        # Spring/Framework detection (common annotations)
        metrics["uses_spring"] = (
            "@RestController" in content or
            "@Service" in content or
            "@Repository" in content or
            "@Component" in content
        )

        # JUnit detection
        metrics["has_junit"] = (
            "@Test" in content or
            "extends TestCase" in content or
            "extends TestCase" in content
        )

        # Main method detection
        metrics["has_main_method"] = any(
            e.name == "main" and "String[]" in str(e.parameters)
            for e in entities
        )

        # POJO detection (simple heuristic)
        is_pojo = any(
            e.type == "class" and (
                "get" in " ".join(e.methods).lower() or
                "set" in " ".join(e.methods).lower()
            )
            for e in entities
        )
        metrics["is_pojo"] = is_pojo

        return metrics
