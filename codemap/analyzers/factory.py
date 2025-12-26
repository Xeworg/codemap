"""Fábrica de Analizadores para CodeMap.

Crea el analizador apropiado para un archivo dado basándose en su lenguaje
o extensión de archivo. Soporta carga diferida de analizadores.

Ejemplo
-------
>>> from codemap.analyzers.factory import AnalyzerFactory
>>> factory = AnalyzerFactory()
>>> analyzer = factory.get_analyzer(Path("example.py"))
>>> result = analyzer.analyze_file(Path("example.py"))
"""

from pathlib import Path
from typing import Dict, Type, Optional, List

from codemap.analyzers.base import BaseAnalyzer
from codemap.parsers import utils as parser_utils


class AnalyzerFactory:
    """Fábrica que crea analizadores específicos de cada lenguaje.

    Proporciona un punto de entrada único para obtener el analizador
    apropiado para cualquier tipo de archivo soportado. Usa carga diferida
    para minimizar las importaciones.

    Atributos:
        _analyzers: Diccionario que mapea nombres de lenguajes a referencias de clases de analizador (como strings)
        _instances: Diccionario de instancias de analizadores en caché
    """

    # Mapping of language -> analyzer class reference (format: "module:classname")
    _analyzers: Dict[str, str] = {}

    # Cached instances
    _instances: Dict[str, BaseAnalyzer] = {}

    def __init__(self):
        """Inicializa la fábrica y registra los analizadores por defecto."""
        self._register_default_analyzers()

    def _register_default_analyzers(self) -> None:
        """Registra todos los analizadores incorporados.

        Usa importación diferida para evitar cargar los lenguajes de tree-sitter
        a menos que realmente se necesiten.
        """
        self._analyzers = {
            "python": "codemap.analyzers.python:PythonAnalyzer",
            "javascript": "codemap.analyzers.javascript:JavaScriptAnalyzer",
            "typescript": "codemap.analyzers.typescript:TypeScriptAnalyzer",
            "java": "codemap.analyzers.java:JavaAnalyzer",
        }

    def get_analyzer_by_language(self, language: str) -> Optional[BaseAnalyzer]:
        """Obtiene una instancia de analizador para un lenguaje específico.

        Args:
            language: Identificador del lenguaje (ej., 'python', 'java').

        Returns:
            Instancia del analizador para el lenguaje, o None si no está soportado.
        """
        language = language.lower()

        # Check cached instance
        if language in self._instances:
            return self._instances[language]

        # Get analyzer class reference
        analyzer_ref = self._analyzers.get(language)
        if not analyzer_ref:
            return None

        # Load analyzer class
        analyzer_class = self._load_analyzer_class(analyzer_ref)
        if not analyzer_class:
            return None

        # Create and cache instance
        instance = analyzer_class()
        self._instances[language] = instance
        return instance

    def get_analyzer_by_file(self, file_path: Path) -> Optional[BaseAnalyzer]:
        """Obtiene el analizador apropiado para un archivo.

        Determina el lenguaje desde la extensión del archivo y retorna
        el analizador correspondiente.

        Args:
            file_path: Ruta del archivo a analizar.

        Returns:
            Instancia del analizador para el archivo, o None si no está soportado.
        """
        # Detect language from extension
        language = parser_utils.detect_language(file_path)
        if not language:
            return None

        return self.get_analyzer_by_language(language)

    def get_analyzer_by_content(self, content: str) -> Optional[BaseAnalyzer]:
        """Infiere el analizador desde el contenido del archivo.

        Examina el contenido para determinar el lenguaje cuando la extensión
        del archivo no está disponible.

        Args:
            content: Contenido del código fuente.

        Returns:
            Instancia del analizador basada en heurísticas del contenido.
        """
        content = content.strip()

        if content.startswith("import ") or content.startswith("from "):
            if "def " in content or "class " in content or "import " in content:
                if "::" not in content and "public class" not in content:
                    return self.get_analyzer_by_language("python")

        if "interface " in content or "class " in content:
            if "public class" in content or "private " in content:
                return self.get_analyzer_by_language("java")

        if "import React" in content or "export default" in content:
            if "interface " in content:
                return self.get_analyzer_by_language("typescript")
            return self.get_analyzer_by_language("javascript")

        if "function" in content or "const " in content or "let " in content:
            return self.get_analyzer_by_language("javascript")

        # Default to Python
        return self.get_analyzer_by_language("python")

    def can_analyze(self, file_path: Path) -> bool:
        """Verifica si algún analizador puede manejar el archivo dado.

        Args:
            file_path: Ruta del archivo.

        Returns:
            True si existe un analizador para este tipo de archivo.
        """
        language = parser_utils.detect_language(file_path)
        return language in self._analyzers

    def get_supported_languages(self) -> List[str]:
        """Obtiene la lista de identificadores de lenguajes soportados.

        Returns:
            Lista de nombres de lenguajes que tienen analizadores.
        """
        return list(self._analyzers.keys())

    def get_supported_extensions(self) -> List[str]:
        """Obtiene la lista de todas las extensiones de archivos soportadas.

        Returns:
            Lista de extensiones de archivos (ej., ['.py', '.java']).
        """
        extensions = set()
        for analyzer_ref in self._analyzers.values():
            analyzer_class = self._load_analyzer_class(analyzer_ref)
            if analyzer_class:
                extensions.update(analyzer_class.SUPPORTED_EXTENSIONS)
        return sorted(list(extensions))

    def _load_analyzer_class(self, analyzer_ref: str) -> Optional[Type[BaseAnalyzer]]:
        """Carga una clase de analizador desde un string de referencia.

        Args:
            analyzer_ref: String en formato 'modulo:nombre_clase'.

        Returns:
            Clase del analizador, o None si la carga falla.
        """
        try:
            module_path, class_name = analyzer_ref.split(":")
            from importlib import import_module

            module = import_module(module_path)
            return getattr(module, class_name)
        except (ValueError, ImportError, AttributeError):
            return None

    def register_analyzer(
        self, language: str, analyzer_class: Type[BaseAnalyzer]
    ) -> None:
        """Registra un nuevo analizador para un lenguaje.

        Args:
            language: Identificador del lenguaje.
            analyzer_class: Clase del analizador a registrar.
        """
        self._analyzers[language] = f"{analyzer_class.__module__}:{analyzer_class.__name__}"
        # Clear cached instance if exists
        if language in self._instances:
            del self._instances[language]

    def unregister_analyzer(self, language: str) -> bool:
        """Elimina el registro de un analizador para un lenguaje.

        Args:
            language: Identificador del lenguaje a eliminar.

        Returns:
            True si se eliminó un analizador, False si no se encontró.
        """
        if language in self._analyzers:
            del self._analyzers[language]
            # Clear cached instance if exists
            if language in self._instances:
                del self._instances[language]
            return True
        return False

    def clear_cache(self) -> None:
        """Borra todas las instancias de analizadores en caché."""
        self._instances.clear()

    def __repr__(self) -> str:
        """Representación en string de la fábrica."""
        return f"AnalyzerFactory(supported_languages={list(self._analyzers.keys())})"
