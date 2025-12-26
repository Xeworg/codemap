"""Parser AST base para CodeMap.

Proporciona clases base abstractas y estructuras de datos para el análisis de código.
Todos los parsers específicos de un lenguaje deben heredar de BaseASTParser.

Estructuras de Datos
--------------------
- Entity: Representa una entidad de código (clase, función, método, etc.)
- CallEdge: Representa una relación de llamada entre entidades
- Dependency: Representa una dependencia de módulo
- ParseResult: Contiene toda la información analizada de un archivo

Ejemplo
-------
>>> from codemap.parsers.ast_parser import BaseASTParser, Entity
>>> class MiParser(BaseASTParser):
...     def parse_file(self, file_path):
...         return ParseResult(file=str(file_path), language="python")
...
>>> parser = MiParser()
"""

from abc import ABC, abstractmethod
from pathlib import Path
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field

import codemap.parsers.utils as parser_utils


@dataclass
class Entity:
    """Representa una entidad de código (clase, función, método, etc.).

    Atributos:
        id: Identificador único en formato "tipo:nombre" (ej: "class:User")
        type: Tipo de entidad (class, function, method, module, etc.)
        name: Nombre de la entidad
        file: Ruta relativa del archivo donde se define la entidad
        line: Número de línea donde inicia la entidad
        end_line: Número de línea donde termina la entidad (opcional)
        parent: ID de la entidad padre si está anidada (opcional)
        children: Lista de IDs de entidades hijo (opcional)
        methods: Lista de nombres de métodos para entidades de clase
        parameters: Lista de nombres de parámetros para funciones/métodos
        decorators: Lista de decoradores aplicados a la entidad
        imports: Lista de módulos/símbolos importados
        docstring: Cadena de documentación si existe
        loc: Líneas de código para esta entidad
        complexity: Complejidad ciclomática (default: 1)
    """

    id: str
    type: str
    name: str
    file: str
    line: int
    end_line: Optional[int] = None
    parent: Optional[str] = None
    children: List[str] = field(default_factory=list)
    methods: List[str] = field(default_factory=list)
    parameters: List[str] = field(default_factory=list)
    decorators: List[str] = field(default_factory=list)
    imports: List[str] = field(default_factory=list)
    docstring: Optional[str] = None
    loc: int = 0
    complexity: int = 1


@dataclass
class CallEdge:
    """Representa una relación de llamada entre entidades.

    Atributos:
        from_id: ID de la entidad que llama
        to_id: ID de la entidad llamada
        call_type: Tipo de llamada (direct_call, callback, inheritance, etc.)
        line: Número de línea donde ocurre la llamada
    """

    from_id: str
    to_id: str
    call_type: str = "direct_call"
    line: int = 0


@dataclass
class Dependency:
    """Representa una dependencia de módulo.

    Atributos:
        source_module: Módulo que tiene la dependencia
        target_module: Módulo del que depende
        symbols: Lista de símbolos específicos importados/usados
        dependency_type: Tipo de dependencia (import, require, inheritance, etc.)
    """

    source_module: str
    target_module: str
    symbols: List[str] = field(default_factory=list)
    dependency_type: str = "import"


@dataclass
class ParseResult:
    """Resultado del análisis de un solo archivo.

    Contiene toda la información extraída incluyendo entidades, llamadas,
    dependencias y métricas.

    Atributos:
        file: Ruta absoluta del archivo analizado
        language: Lenguaje de programación detectado
        entities: Lista de entidades extraídas
        calls: Lista de relaciones de llamada
        dependencies: Lista de dependencias de módulos
        metrics: Diccionario de métricas calculadas
        errors: Lista de mensajes de error encontrados durante el análisis

    Ejemplo:
        >>> resultado = parser.parse_file(Path("user.py"))
        >>> len(resultado.entidades)
        5
        >>> resultado.metrics["total_loc"]
        150
    """

    file: str
    language: str
    entities: List[Entity] = field(default_factory=list)
    calls: List[CallEdge] = field(default_factory=list)
    dependencies: List[Dependency] = field(default_factory=list)
    metrics: Dict[str, Any] = field(default_factory=dict)
    errors: List[str] = field(default_factory=list)


class BaseASTParser(ABC):
    """Clase base abstracta para parsers de AST.

    Todos los parsers específicos de un lenguaje deben heredar de esta clase
    e implementar los métodos requeridos.

    Las subclases deben:
    1. Definir el atributo de clase SUPPORTED_LANGUAGES
    2. Implementar parse_file() para analizar desde ruta de archivo
    3. Implementar parse_content() para analizar desde contenido string

    Atributos:
        entity_counter: Contador interno para generar IDs únicos de entidades.

    Ejemplo:
        >>> class PythonParser(BaseASTParser):
        ...     SUPPORTED_LANGUAGES = ["python"]
        ...
        ...     def parse_file(self, file_path):
        ...         content = self.read_file(file_path)
        ...         return self.parse_content(content, file_path)
        ...
        ...     def parse_content(self, content, file_path):
        ...         # Implementación aquí
        ...         return ParseResult(...)
    """

    SUPPORTED_LANGUAGES: List[str] = []

    def __init__(self):
        self.entity_counter = 0

    @abstractmethod
    def parse_file(self, file_path: Path) -> ParseResult:
        """Analiza un solo archivo y devuelve entidades, llamadas y dependencias.

        Args:
            file_path: Ruta al archivo a analizar.

        Returns:
            ParseResult con toda la información extraída.

        Raises:
            NotImplementedError: Debe ser implementado por la subclase.
        """
        pass

    @abstractmethod
    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Analiza contenido de archivo directamente sin leer del disco.

        Args:
            content: Contenido del archivo como string.
            file_path: Objeto Path para referencia (puede no existir).

        Returns:
            ParseResult con toda la información extraída.

        Raises:
            NotImplementedError: Debe ser implementado por la subclase.
        """
        pass

    def generate_entity_id(self, entity_type: str, name: str) -> str:
        """Genera un ID único para una entidad.

        Usa el formato "tipo:nombre" con un contador incremental.

        Args:
            entity_type: Tipo de entidad (class, function, method, etc.)
            name: Nombre de la entidad.

        Returns:
            Identificador único como string.

        Ejemplo:
            >>> self.generate_entity_id("class", "User")
            'class:User'
            >>> self.generate_entity_id("function", "save")
            'function:save'
        """
        self.entity_counter += 1
        return f"{entity_type}:{name}"

    def read_file(self, file_path: Path) -> Optional[str]:
        """Lee el contenido de un archivo de forma segura.

        Args:
            file_path: Ruta al archivo a leer.

        Returns:
            Contenido del archivo como string, o None si falla la lectura.
        """
        return parser_utils.read_file_safe(file_path)

    def extract_docstring(self, content: str, start_line: int) -> Optional[str]:
        """Extrae docstring del contenido comenzando en una línea.

        Args:
            content: Contenido completo del archivo.
            start_line: Número de línea (1-indexed) donde comenzar a buscar.

        Returns:
            Texto del docstring si se encuentra, None en caso contrario.
        """
        lines = content.split("\n")
        if start_line - 1 < len(lines):
            line = lines[start_line - 1].strip()
            if line.startswith('"""') or line.startswith("'''"):
                end_marker = line[3:]
                if end_marker and not end_marker.startswith("\\"):
                    return line[3:-3]
        return None
