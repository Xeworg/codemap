"""Analizadores de CodeMap.

Analizadores de código específicos de cada lenguaje para extraer entidades,
construir grafos de llamadas y calcular métricas de calidad.

Módulos
-------
base
    Clases base y tipos de resultados para todos los analizadores.
python
    Analizador del lenguaje Python usando libcst.
javascript
    Analizador del lenguaje JavaScript usando tree-sitter.
typescript
    Analizador del lenguaje TypeScript usando tree-sitter.
java
    Analizador del lenguaje Java usando tree-sitter.
factory
    Fábrica para crear analizadores específicos de cada lenguaje.

Ejemplo
-------
>>> from codemap.analyzers.factory import AnalyzerFactory
>>> from pathlib import Path
>>> factory = AnalyzerFactory()
>>> analyzer = factory.get_analyzer_by_file(Path("example.py"))
>>> result = analyzer.analyze_file(Path("example.py"))
"""

from pathlib import Path

from codemap.analyzers.base import (
    BaseAnalyzer,
    AnalysisResult,
    ProjectAnalysisResult,
)

from codemap.analyzers.factory import AnalyzerFactory

__all__ = [
    "BaseAnalyzer",
    "AnalysisResult",
    "ProjectAnalysisResult",
    "AnalyzerFactory",
]

# Global factory instance
_factory: AnalyzerFactory | None = None


def get_factory() -> AnalyzerFactory:
    """Obtiene la instancia global de la fábrica de analizadores.

    Returns:
        Instancia singleton de AnalyzerFactory.
    """
    global _factory
    if _factory is None:
        _factory = AnalyzerFactory()
    return _factory


def get_analyzer(file_path: str | Path):
    """Obtiene el analizador apropiado para un archivo.

    Args:
        file_path: Ruta del archivo a analizar.

    Returns:
        Instancia del analizador para el tipo de archivo.
    """
    return get_factory().get_analyzer_by_file(Path(file_path))


def analyze_file(file_path: str | Path):
    """Analiza un solo archivo usando el analizador apropiado.

    Args:
        file_path: Ruta del archivo a analizar.

    Returns:
        AnalysisResult con entidades y métricas extraídas.
    """
    analyzer = get_analyzer(file_path)
    if not analyzer:
        raise ValueError(f"No analyzer available for: {file_path}")
    return analyzer.analyze_file(Path(file_path))
