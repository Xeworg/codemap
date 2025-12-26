"""Analizador de Python para CodeMap.

Analiza archivos fuente de Python para extraer entidades, construir grafos de llamadas
y calcular métricas de calidad.

Ejemplo
-------
>>> from codemap.analyzers.python import PythonAnalyzer
>>> analyzer = PythonAnalyzer()
>>> result = analyzer.analyze_file(Path("example.py"))
>>> len(result.entities)
5
"""

from pathlib import Path
from typing import List, Dict, Any

from codemap.parsers.ast_parser import BaseASTParser
from codemap.parsers.python_parser import PythonASTParser
from codemap.analyzers.base import BaseAnalyzer, AnalysisResult
from codemap.parsers import utils as parser_utils


class PythonAnalyzer(BaseAnalyzer):
    """Analizador del lenguaje Python.

    Maneja archivos .py y .pyi, extrayendo clases, funciones, métodos,
    importaciones y relaciones de llamadas usando libcst para un parseo preciso.

    Atributos:
        SUPPORTED_EXTENSIONS: Extensiones de archivos que maneja este analizador
        SUPPORTED_LANGUAGES: Identificadores de lenguajes que soporta este analizador
    """

    SUPPORTED_EXTENSIONS = [".py", ".pyi", ".pyx", ".pxd", ".pyw"]
    SUPPORTED_LANGUAGES = ["python"]

    def _create_parser(self) -> BaseASTParser:
        """Crea un parser AST de Python.

        Returns:
            Instancia de PythonASTParser.
        """
        return PythonASTParser()

    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si este analizador puede manejar el archivo Python dado.

        Args:
            file_path: Ruta del archivo.

        Returns:
            True si la extensión es .py, .pyi, .pyx, .pxd, o .pyw.
        """
        return parser_utils.detect_language(file_path) == "python"

    def _extract_metrics(self, content: str, entities: List) -> Dict[str, Any]:
        """Calcula métricas de calidad específicas de Python.

        Args:
            content: Código fuente de Python.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario de métricas calculadas.
        """
        metrics = {}

        # Basic counts
        lines = content.split("\n")
        metrics["total_lines"] = len(lines)
        metrics["non_empty_lines"] = len([l for l in lines if l.strip()])

        # Code lines (excluding comments)
        metrics["code_lines"] = parser_utils.count_lines_of_code(content)

        # Comment lines
        comment_lines = 0
        for line in lines:
            stripped = line.strip()
            if stripped.startswith("#") or stripped.startswith('"""') or stripped.startswith("'''"):
                comment_lines += 1
        metrics["comment_lines"] = comment_lines

        # Comment ratio
        if metrics["total_lines"] > 0:
            metrics["comment_ratio"] = round(
                metrics["comment_lines"] / metrics["total_lines"], 2
            )

        # Entity-based metrics
        classes = [e for e in entities if e.type == "class"]
        functions = [e for e in entities if e.type == "function"]
        methods = [e for e in entities if e.type == "method"]

        metrics["class_count"] = len(classes)
        metrics["function_count"] = len(functions)
        metrics["method_count"] = len(methods)

        # Complexity metrics
        if entities:
            max_complexity = max(e.complexity for e in entities)
            avg_complexity = sum(e.complexity for e in entities) / len(entities)
            metrics["max_complexity"] = max_complexity
            metrics["avg_complexity"] = round(avg_complexity, 2)

            # Find hotspots (high complexity entities)
            hotspots = [
                {"name": e.name, "complexity": e.complexity, "type": e.type}
                for e in entities
                if e.complexity > 10
            ]
            metrics["complexity_hotspots"] = sorted(
                hotspots, key=lambda x: x["complexity"], reverse=True
            )[:5]

        # Large classes/functions (potential refactoring targets)
        refactoring_targets = []
        for entity in entities:
            if entity.type == "class" and len(entity.methods) > 10:
                refactoring_targets.append(
                    {
                        "name": entity.name,
                        "type": "class",
                        "reason": f"Too many methods ({len(entity.methods)})",
                        "file": entity.file,
                        "line": entity.line,
                    }
                )
            if entity.type == "function" and entity.parameters and len(entity.parameters) > 5:
                refactoring_targets.append(
                    {
                        "name": entity.name,
                        "type": "function",
                        "reason": f"Too many parameters ({len(entity.parameters)})",
                        "file": entity.file,
                        "line": entity.line,
                    }
                )

        metrics["refactoring_targets"] = refactoring_targets[:5]

        # Inheritance depth (simplified)
        inheritance_depths = []
        for entity in entities:
            if entity.type == "class" and entity.parent:
                inheritance_depths.append(entity)
        metrics["inheritance_depth"] = len(inheritance_depths)

        # Static/Class methods detection (simplified heuristic)
        static_methods = []
        for entity in entities:
            if entity.type == "method":
                if "staticmethod" in entity.decorators or entity.name.startswith("_") and not entity.name.startswith("__"):
                    pass  # Heuristic: private methods often static

        # Public API estimation
        public_entities = [
            e for e in entities if not e.name.startswith("_")
        ]
        metrics["public_api_count"] = len(public_entities)

        return metrics

    def analyze_file(self, file_path: Path) -> AnalysisResult:
        """Analiza un archivo Python con métricas mejoradas.

        Args:
            file_path: Ruta del archivo Python.

        Returns:
            AnalysisResult con métricas específicas de Python mejoradas.
        """
        result = super().analyze_file(file_path)

        # Add Python-specific analysis
        if result.success:
            # Enhanced import analysis
            imports = [d for d in result.dependencies if d.dependency_type == "import"]
            result.metrics["import_count"] = len(imports)

            # Count stdlib vs third-party vs local
            stdlib_imports = 0
            third_party = 0
            local_imports = 0

            for dep in imports:
                module = dep.target_module
                if module.startswith("."):
                    local_imports += 1
                elif "." in module:
                    third_party += 1
                else:
                    stdlib_imports += 1

            result.metrics["stdlib_imports"] = stdlib_imports
            result.metrics["third_party_imports"] = third_party
            result.metrics["local_imports"] = local_imports

            # Module naming convention check
            if file_path.name.startswith("_"):
                result.metrics["private_module"] = True

        return result
