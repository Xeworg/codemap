"""Base AST Parser for CodeMap.

Provides abstract base classes and data structures for code parsing.
All language-specific parsers should inherit from BaseASTParser.

Data Structures
---------------
- Entity: Represents a code entity (class, function, method, etc.)
- CallEdge: Represents a call relationship between entities
- Dependency: Represents a module dependency
- ParseResult: Contains all parsed information from a file

Example
-------
>>> from codemap.parsers.ast_parser import BaseASTParser, Entity
>>> class MyParser(BaseASTParser):
...     def parse_file(self, file_path):
...         return ParseResult(file=str(file_path), language="python")
...
>>> parser = MyParser()
"""

from abc import ABC, abstractmethod
from pathlib import Path
from typing import List, Dict, Any, Optional
from dataclasses import dataclass, field

import codemap.parsers.utils as parser_utils


@dataclass
class Entity:
    """Represents a code entity (class, function, method, etc.).

    Attributes:
        id: Unique identifier in format "type:name" (e.g., "class:User")
        type: Entity type (class, function, method, module, etc.)
        name: Name of the entity
        file: Relative file path where entity is defined
        line: Line number where entity starts
        end_line: Line number where entity ends (optional)
        parent: ID of parent entity if nested (optional)
        children: List of child entity IDs (optional)
        methods: List of method names for class entities
        parameters: List of parameter names for functions/methods
        decorators: List of decorator names applied to entity
        imports: List of imported modules/symbols
        docstring: Documentation string if present
        loc: Lines of code for this entity
        complexity: Cyclomatic complexity (default: 1)
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
    """Represents a call relationship between entities.

    Attributes:
        from_id: ID of the calling entity
        to_id: ID of the called entity
        call_type: Type of call (direct_call, callback, inheritance, etc.)
        line: Line number where the call occurs
    """

    from_id: str
    to_id: str
    call_type: str = "direct_call"
    line: int = 0


@dataclass
class Dependency:
    """Represents a module dependency.

    Attributes:
        source_module: Module that has the dependency
        target_module: Module being depended upon
        symbols: List of specific symbols imported/used
        dependency_type: Type of dependency (import, require, inheritance, etc.)
    """

    source_module: str
    target_module: str
    symbols: List[str] = field(default_factory=list)
    dependency_type: str = "import"


@dataclass
class ParseResult:
    """Result of parsing a single file.

    Contains all extracted information including entities, calls,
    dependencies, and metrics.

    Attributes:
        file: Absolute path to the parsed file
        language: Detected programming language
        entities: List of extracted entities
        calls: List of call relationships
        dependencies: List of module dependencies
        metrics: Dictionary of computed metrics
        errors: List of error messages encountered during parsing

    Example:
        >>> result = parser.parse_file(Path("user.py"))
        >>> len(result.entities)
        5
        >>> result.metrics["total_loc"]
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
    """Abstract base class for AST parsers.

    All language-specific parsers must inherit from this class
    and implement the required methods.

    Subclasses should:
    1. Set SUPPORTED_LANGUAGES class attribute
    2. Implement parse_file() to parse from file path
    3. Implement parse_content() to parse from string content

    Attributes:
        entity_counter: Internal counter for generating unique entity IDs.

    Example:
        >>> class PythonParser(BaseASTParser):
        ...     SUPPORTED_LANGUAGES = ["python"]
        ...
        ...     def parse_file(self, file_path):
        ...         content = self.read_file(file_path)
        ...         return self.parse_content(content, file_path)
        ...
        ...     def parse_content(self, content, file_path):
        ...         # Implementation here
        ...         return ParseResult(...)
    """

    SUPPORTED_LANGUAGES: List[str] = []

    def __init__(self):
        self.entity_counter = 0

    @abstractmethod
    def parse_file(self, file_path: Path) -> ParseResult:
        """Parse a single file and return entities, calls, and dependencies.

        Args:
            file_path: Path to the file to parse.

        Returns:
            ParseResult containing all extracted information.

        Raises:
            NotImplementedError: Must be implemented by subclass.
        """
        pass

    @abstractmethod
    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Parse file content directly without reading from disk.

        Args:
            content: Raw file content as string.
            file_path: Path object for reference (may not exist).

        Returns:
            ParseResult containing all extracted information.

        Raises:
            NotImplementedError: Must be implemented by subclass.
        """
        pass

    def generate_entity_id(self, entity_type: str, name: str) -> str:
        """Generate a unique entity ID.

        Uses format "type:name" with an incrementing counter.

        Args:
            entity_type: Type of entity (class, function, method, etc.)
            name: Name of the entity.

        Returns:
            Unique identifier string.

        Example:
            >>> self.generate_entity_id("class", "User")
            'class:User'
            >>> self.generate_entity_id("function", "save")
            'function:save'
        """
        self.entity_counter += 1
        return f"{entity_type}:{name}"

    def read_file(self, file_path: Path) -> Optional[str]:
        """Read file content safely using parser utilities.

        Args:
            file_path: Path to the file to read.

        Returns:
            File contents as string, or None if reading fails.
        """
        return parser_utils.read_file_safe(file_path)

    def extract_docstring(self, content: str, start_line: int) -> Optional[str]:
        """Extract docstring from content starting at line.

        Args:
            content: Full file content.
            start_line: Line number (1-indexed) to start searching.

        Returns:
            Docstring text if found, None otherwise.
        """
        lines = content.split("\n")
        if start_line - 1 < len(lines):
            line = lines[start_line - 1].strip()
            if line.startswith('"""') or line.startswith("'''"):
                end_marker = line[3:]
                if end_marker and not end_marker.startswith("\\"):
                    return line[3:-3]
        return None
