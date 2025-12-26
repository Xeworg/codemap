"""Analizador base para CodeMap.

Proporciona la clase base abstracta de la que deben heredar todos los analizadores
específicos de cada lenguaje. Maneja la lógica común de análisis, extracción de
entidades y construcción de grafos de llamadas.

Ejemplo
-------
>>> from codemap.analyzers.base import BaseAnalyzer
>>> class PythonAnalyzer(BaseAnalyzer):
...     SUPPORTED_EXTENSIONS = [".py", ".pyi"]
...
>>> analyzer = PythonAnalyzer()
>>> result = analyzer.analyze_file(Path("example.py"))
"""

from abc import ABC, abstractmethod
from pathlib import Path
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field

from codemap.parsers.ast_parser import (
    BaseASTParser,
    ParseResult,
    Entity,
    CallEdge,
    Dependency,
)


@dataclass
class AnalysisResult:
    """Resultado completo del análisis de un archivo o proyecto.

    Atributos:
        file: Ruta del archivo que fue analizado
        success: Si el análisis se completó exitosamente
        entities: Lista de entidades extraídas (clases, funciones, etc.)
        calls: Lista de relaciones de llamadas entre entidades
        dependencies: Lista de dependencias de módulos
        metrics: Diccionario de métricas calculadas
        errors: Lista de mensajes de error si los hay
    """
    file: Path
    success: bool = True
    entities: List[Entity] = field(default_factory=list)
    calls: List[CallEdge] = field(default_factory=list)
    dependencies: List[Dependency] = field(default_factory=list)
    metrics: Dict[str, Any] = field(default_factory=dict)
    errors: List[str] = field(default_factory=list)


@dataclass
class ProjectAnalysisResult:
    """Resultado completo del análisis de un proyecto completo.

    Atributos:
        project_path: Ruta raíz del proyecto analizado
        files_analyzed: Número de archivos analizados exitosamente
        total_entities: Total de entidades encontradas en todos los archivos
        total_calls: Total de relaciones de llamadas encontradas
        total_dependencies: Total de dependencias de módulos encontradas
        file_results: Resultados individuales por archivo
        project_metrics: Métricas agregadas del proyecto
        errors: Lista de errores durante el análisis
    """
    project_path: Path
    files_analyzed: int = 0
    total_entities: int = 0
    total_calls: int = 0
    total_dependencies: int = 0
    file_results: List[AnalysisResult] = field(default_factory=list)
    project_metrics: Dict[str, Any] = field(default_factory=dict)
    errors: List[str] = field(default_factory=list)


class BaseAnalyzer(ABC):
    """Clase base abstracta para todos los analizadores de lenguajes.

    Todos los analizadores específicos de cada lenguaje deben heredar de esta clase e implementar
    los métodos abstractos requeridos.

    El analizador es responsable de:
    1. Parsear archivos fuente usando parsers específicos del lenguaje
    2. Extraer entidades (clases, funciones, métodos)
    3. Construir grafos de llamadas (quién llama a quién)
    4. Rastrear dependencias de módulos
    5. Calcular métricas de calidad

    Atributos:
        SUPPORTED_EXTENSIONS: Lista de extensiones de archivos que maneja este analizador
        SUPPORTED_LANGUAGES: Lista de nombres de lenguajes que soporta este analizador
        parser: Instancia del parser AST para este lenguaje

    Ejemplo:
        >>> class PythonAnalyzer(BaseAnalyzer):
        ...     SUPPORTED_EXTENSIONS = [".py"]
        ...     SUPPORTED_LANGUAGES = ["python"]
        ...
        ...     def _create_parser(self) -> BaseASTParser:
        ...         return PythonASTParser()
    """

    # Subclasses must override these
    SUPPORTED_EXTENSIONS: List[str] = []
    SUPPORTED_LANGUAGES: List[str] = []

    def __init__(self):
        """Inicializa el analizador con un parser específico del lenguaje."""
        self.parser: BaseASTParser = self._create_parser()

    @abstractmethod
    def _create_parser(self) -> BaseASTParser:
        """Crea y retorna un parser AST específico del lenguaje.

        Returns:
            Una instancia de un parser para este lenguaje.
        """
        pass

    @abstractmethod
    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si este analizador puede procesar el archivo dado.

        Args:
            file_path: Ruta del archivo a verificar.

        Returns:
            True si este analizador puede manejar el archivo, False en caso contrario.
        """
        pass

    @abstractmethod
    def _extract_metrics(self, content: str, entities: List[Entity]) -> Dict[str, Any]:
        """Calcula métricas de calidad para el contenido analizado.

        Args:
            content: El contenido del código fuente.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario conteniendo las métricas calculadas.
        """
        pass

    def analyze_file(self, file_path: Path) -> AnalysisResult:
        """Analiza un único archivo fuente.

        Args:
            file_path: Ruta del archivo a analizar.

        Returns:
            AnalysisResult conteniendo entidades, llamadas, dependencias y métricas.
        """
        result = AnalysisResult(file=file_path)

        try:
            # Parse the file
            parse_result = self.parser.parse_file(file_path)

            # Extract entities
            result.entities = parse_result.entities
            result.calls = parse_result.calls
            result.dependencies = parse_result.dependencies

            # Read content for metrics
            content = self.parser.read_file(file_path)
            if content:
                # Compute metrics
                result.metrics = self._extract_metrics(content, result.entities)
                result.metrics["total_loc"] = content.count("\n")
                result.metrics["file_size"] = file_path.stat().st_size

            result.success = True

        except Exception as e:
            result.success = False
            result.errors.append(f"Error analyzing {file_path}: {str(e)}")

        return result

    def analyze_content(
        self, content: str, file_path: Path, file_language: str = ""
    ) -> AnalysisResult:
        """Analiza código fuente desde un string.

        Args:
            content: Código fuente como string.
            file_path: Ruta para simular la ubicación del archivo.
            file_language: Sugerencia de lenguaje para el contenido.

        Returns:
            AnalysisResult conteniendo entidades, llamadas, dependencias y métricas.
        """
        result = AnalysisResult(file=file_path)

        try:
            # Parse the content
            parse_result = self.parser.parse_content(content, file_path)

            # Extract entities
            result.entities = parse_result.entities
            result.calls = parse_result.calls
            result.dependencies = parse_result.dependencies

            # Compute metrics
            result.metrics = self._extract_metrics(content, result.entities)
            result.metrics["total_loc"] = content.count("\n")

            result.success = True

        except Exception as e:
            result.success = False
            result.errors.append(f"Error analyzing content: {str(e)}")

        return result

    def _build_call_graph_stats(self, calls: List[CallEdge]) -> Dict[str, Any]:
        """Construye estadísticas sobre la conectividad del grafo de llamadas.

        Args:
            calls: Lista de relaciones de llamadas.

        Returns:
            Diccionario conteniendo estadísticas del grafo de llamadas.
        """
        if not calls:
            return {
                "total_calls": 0,
                "unique_callers": 0,
                "unique_callees": 0,
                "max_call_depth": 0,
                "hotspots": [],
            }

        # Count callers and callees
        callers = set(call.from_id for call in calls)
        callees = set(call.to_id for call in calls)

        # Count calls per entity
        call_counts: Dict[str, int] = {}
        for call in calls:
            call_counts[call.from_id] = call_counts.get(call.from_id, 0) + 1

        # Find hotspots (most called entities)
        callee_counts: Dict[str, int] = {}
        for call in calls:
            callee_counts[call.to_id] = callee_counts.get(call.to_id, 0) + 1

        hotspots = sorted(
            [{"entity": k, "calls": v} for k, v in callee_counts.items()],
            key=lambda x: x["calls"],
            reverse=True,
        )[:10]

        return {
            "total_calls": len(calls),
            "unique_callers": len(callers),
            "unique_callees": len(callees),
            "hotspots": hotspots,
        }

    def _get_entity_summary(self, entities: List[Entity]) -> Dict[str, Any]:
        """Genera estadísticas de resumen para las entidades.

        Args:
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario conteniendo estadísticas de entidades.
        """
        by_type: Dict[str, int] = {}
        for entity in entities:
            by_type[entity.type] = by_type.get(entity.type, 0) + 1

        return {
            "total": len(entities),
            "by_type": by_type,
            "classes_with_many_methods": [
                e.name
                for e in entities
                if e.type == "class" and len(e.methods) > 10
            ],
        }
