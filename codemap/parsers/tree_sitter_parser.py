"""Wrapper de parser tree-sitter para CodeMap.

Proporciona capacidades de análisis multi-lenguaje usando tree-sitter.
Soporta Python, JavaScript, TypeScript, Java, Go, Rust, C++, Ruby y PHP.

Características
---------------
- Detección de lenguaje desde extensión de archivo
- Extracción de entidades (clases, funciones, métodos)
- Construcción de call graph
- Análisis de dependencias
- Cálculo básico de métricas

Ejemplo
-------
>>> from pathlib import Path
>>> from codemap.parsers import TreeSitterParser
>>> parser = TreeSitterParser()
>>> resultado = parser.parse_file(Path("ejemplo.py"))
>>> for entidad in resultado.entidades:
...     print(f"{entidad.tipo}: {entidad.nombre}")

Dependencias
------------
Requiere tree-sitter y los paquetes de lenguajes:
    pip install tree-sitter
    pip install tree-sitter-python tree-sitter-javascript tree-sitter-typescript
"""

from pathlib import Path
from typing import List, Optional, Dict, Any
from dataclasses import dataclass, field

from tree_sitter import Language, Parser

from codemap.parsers.ast_parser import (
    BaseASTParser,
    Entity,
    CallEdge,
    Dependency,
    ParseResult,
)
from codemap.parsers import utils as parser_utils


@dataclass
class TreeSitterEntity(Entity):
    """Entidad extraída del AST de tree-sitter.

    Extiende Entity con campos específicos de tree-sitter.

    Atributos:
        node_type: Tipo de nodo raw de tree-sitter (ej: 'function_definition')
        is_definition: True si es un nodo de definición
    """

    node_type: str = ""
    is_definition: bool = False


class TreeSitterParser(BaseASTParser):
    """Parser multi-lenguaje usando tree-sitter.

    Analiza archivos de código fuente y extrae entidades, llamadas y dependencias
    usando la representación de AST agnóstico del lenguaje de tree-sitter.

    Lenguajes Soportados
    --------------------
    - Python (.py)
    - JavaScript (.js, .jsx)
    - TypeScript (.ts, .tsx)
    - Java (.java)
    - Go (.go)
    - Rust (.rs)
    - C/C++ (.c, .cpp, .h)
    - Ruby (.rb)
    - PHP (.php)

    Atributos:
        parsers: Diccionario que mapea nombres de lenguaje a parsers de tree-sitter.

    Ejemplo:
        >>> parser = TreeSitterParser()
        >>> resultado = parser.parse_file(Path("main.py"))
        >>> print(f"Encontradas {len(resultado.entidades)} entidades")
    """

    LANGUAGE_MODULES: Dict[str, str] = {
        "python": "tree_sitter_python",
        "javascript": "tree_sitter_javascript",
        "typescript": "tree_sitter_typescript",
        "java": "tree_sitter_java",
        "go": "tree_sitter_go",
        "rust": "tree_sitter_rust",
        "cpp": "tree_sitter_cpp",
        "c": "tree_sitter_c",
        "ruby": "tree_sitter_ruby",
        "php": "tree_sitter_php",
    }

    DEFINITION_TYPES = {
        "function_definition",
        "class_definition",
        "method_definition",
        "function_declaration",
        "class_declaration",
    }

    CALL_TYPES = {
        "call_expression",
        "method_call",
        "identifier",
    }

    IMPORT_TYPES = {
        "import_statement",
        "import_from_statement",
        "require_call",
    }

    def __init__(self):
        """Inicializa el parser y carga los parsers de lenguaje."""
        super().__init__()
        self.parsers: Dict[str, Parser] = {}
        self._load_languages()

    def _load_languages(self):
        """Carga los parsers de lenguaje de tree-sitter.

        Requiere instalar los paquetes de lenguajes:
        pip install tree-sitter-python tree-sitter-javascript tree-sitter-typescript
        """
        for lang, module_name in self.LANGUAGE_MODULES.items():
            try:
                module = __import__(module_name, fromlist=["language"])
                language = Language(module.language())
                self.parsers[lang] = Parser(language)
            except (ImportError, AttributeError):
                pass

    def is_available(self) -> bool:
        """Verifica si hay parsers de lenguaje cargados.

        Returns:
            True si al menos un parser de lenguaje está cargado.
        """
        return len(self.parsers) > 0

    def parse_file(self, file_path: Path) -> ParseResult:
        """Analiza un archivo usando tree-sitter.

        Args:
            file_path: Ruta al archivo a analizar.

        Returns:
            ParseResult con entidades, llamadas, dependencias y métricas extraídas.
        """
        content = self.read_file(file_path)
        if content is None:
            return ParseResult(
                file=str(file_path), language="unknown", errors=["No se pudo leer el archivo"]
            )
        return self.parse_content(content, file_path)

    def parse_content(self, content: str, file_path: Path) -> ParseResult:
        """Analiza contenido de archivo usando tree-sitter.

        Args:
            content: Contenido del código fuente.
            file_path: Objeto Path para referencia (detección de lenguaje).

        Returns:
            ParseResult con entidades, llamadas, dependencias y métricas extraídas.
        """
        result = ParseResult(
            file=str(file_path),
            language=parser_utils.detect_language(file_path) or "unknown",
        )

        parser = self.parsers.get(result.language)
        if parser is None:
            result.errors.append(f"No hay parser para el lenguaje: {result.language}")
            return result

        tree = parser.parse(bytes(content, "utf-8"))
        result.entities = self._extract_entities(tree.root_node, content, file_path)
        result.calls = self._extract_calls(tree.root_node, content)
        result.dependencies = self._extract_dependencies(tree.root_node, content)
        result.metrics = self._compute_metrics(content, result.entities)

        return result

    def _extract_entities(self, node, content: str, file_path: Path) -> List[TreeSitterEntity]:
        """Extrae definiciones de entidades del AST.

        Recorre recursivamente el AST para encontrar todos los nodos de
        definición (funciones, clases, métodos).

        Args:
            node: Nodo actual de tree-sitter.
            content: Contenido del código fuente para extracción de líneas.
            file_path: Ruta del archivo para metadatos de entidad.

        Returns:
            Lista de objetos TreeSitterEntity extraídos.
        """
        entities = []
        lines = content.split("\n")

        for child in node.children:
            if child.type in self.DEFINITION_TYPES:
                entity = self._node_to_entity(child, content, file_path)
                if entity:
                    entities.append(entity)

            entities.extend(self._extract_entities(child, content, file_path))

        return entities

    def _node_to_entity(self, node, content: str, file_path: Path) -> Optional[TreeSitterEntity]:
        """Convierte un nodo de tree-sitter a una Entidad.

        Args:
            node: Nodo de tree-sitter a convertir.
            content: Contenido del código fuente.
            file_path: Ruta del archivo para metadatos de entidad.

        Returns:
            TreeSitterEntity si la conversión fue exitosa, None en caso contrario.
        """
        name = self._extract_name(node, content)
        if not name:
            return None

        line = node.start_point[0] + 1
        end_line = node.end_point[0] + 1
        loc = end_line - line

        entity_type = self._map_node_type(node.type)

        entity_id = self.generate_entity_id(entity_type, name)

        docstring = self._extract_docstring_from_node(node, content)

        return TreeSitterEntity(
            id=entity_id,
            type=entity_type,
            name=name,
            file=str(file_path),
            line=line,
            end_line=end_line,
            loc=loc,
            docstring=docstring,
            node_type=node.type,
            is_definition=True,
        )

    def _extract_name(self, node, content: str) -> Optional[str]:
        """Extrae el nombre de la entidad del nodo.

        Args:
            node: Nodo de tree-sitter.
            content: Contenido del código fuente.

        Returns:
            Nombre de la entidad si se encuentra, None en caso contrario.
        """
        lines = content.split("\n")

        for child in node.children:
            if child.type == "identifier":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()
            if child.type == "name":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    return lines[line_idx].strip()

        return None

    def _map_node_type(self, node_type: str) -> str:
        """Mapea tipos de nodo de tree-sitter a tipos de entidad.

        Args:
            node_type: String del tipo de nodo raw de tree-sitter.

        Returns:
            Tipo de entidad mapeado (class, function, method, etc.)
        """
        type_map = {
            "function_definition": "function",
            "function_declaration": "function",
            "class_definition": "class",
            "class_declaration": "class",
            "method_definition": "method",
            "method_declaration": "method",
        }
        return type_map.get(node_type, "function")

    def _extract_docstring_from_node(self, node, content: str) -> Optional[str]:
        """Extrae docstring del nodo.

        Args:
            node: Nodo de tree-sitter.
            content: Contenido del código fuente.

        Returns:
            Texto del docstring si se encuentra, None en caso contrario.
        """
        lines = content.split("\n")
        start_line = node.start_point[0]

        if start_line + 1 < len(lines):
            first_line = lines[start_line].strip()
            if first_line.startswith('"""') or first_line.startswith("'''"):
                end_char = first_line[3:]
                if end_char and not end_char.endswith("\\"):
                    return first_line[3:-3]

        return None

    def _extract_calls(self, node, content: str) -> List[CallEdge]:
        """Extrae llamadas a funciones del AST.

        Args:
            node: Nodo actual de tree-sitter.
            content: Contenido del código fuente.

        Returns:
            Lista de objetos CallEdge representando las llamadas a funciones.
        """
        calls = []

        for child in node.children:
            if child.type in self.CALL_TYPES:
                call = self._node_to_call(child, content)
                if call:
                    calls.append(call)

            calls.extend(self._extract_calls(child, content))

        return calls

    def _node_to_call(self, node, content: str) -> Optional[CallEdge]:
        """Convierte un nodo de llamada a un CallEdge.

        Args:
            node: Nodo de tree-sitter representando una llamada.
            content: Contenido del código fuente.

        Returns:
            CallEdge si la conversión fue exitosa, None en caso contrario.
        """
        lines = content.split("\n")
        line = node.start_point[0] + 1

        if node.children:
            callee = node.children[0]
            if callee.type == "identifier":
                line_idx = callee.start_point[0]
                if line_idx < len(lines):
                    callee_name = lines[line_idx].strip()
                    return CallEdge(
                        from_id="",
                        to_id=f"func:{callee_name}",
                        line=line,
                    )

        return None

    def _extract_dependencies(self, node, content: str) -> List[Dependency]:
        """Extrae dependencias de módulos del AST.

        Args:
            node: Nodo actual de tree-sitter.
            content: Contenido del código fuente.

        Returns:
            Lista de objetos Dependency.
        """
        deps = []

        for child in node.children:
            if child.type in self.IMPORT_TYPES:
                dep = self._node_to_dependency(child, content)
                if dep:
                    deps.append(dep)

            deps.extend(self._extract_dependencies(child, content))

        return deps

    def _node_to_dependency(self, node, content: str) -> Optional[Dependency]:
        """Convierte nodo de import a Dependency.

        Args:
            node: Nodo de tree-sitter representando un import.
            content: Contenido del código fuente.

        Returns:
            Dependency si la conversión fue exitosa, None en caso contrario.
        """
        lines = content.split("\n")

        for child in node.children:
            if child.type == "string":
                line_idx = child.start_point[0]
                if line_idx < len(lines):
                    module = lines[line_idx].strip().strip('"').strip("'")
                    return Dependency(
                        source_module="",
                        target_module=module,
                        dependency_type="import",
                    )

        return None

    def _compute_metrics(self, content: str, entities: List[Entity]) -> Dict[str, Any]:
        """Calcula métricas para el contenido analizado.

        Args:
            content: Contenido del código fuente.
            entities: Lista de entidades extraídas.

        Returns:
            Diccionario conteniendo métricas:
            - total_loc: Total de líneas incluyendo vacías/comentarios
            - code_loc: Líneas de código real
            - entity_count: Número de entidades encontradas
            - total_complexity: Suma de complejidades de entidades
            - avg_complexity: Complejidad promedio por entidad
        """
        total_loc = parser_utils.count_lines(content)
        code_loc = parser_utils.count_lines_of_code(content)

        total_complexity = sum(e.complexity for e in entities)

        return {
            "total_loc": total_loc,
            "code_loc": code_loc,
            "entity_count": len(entities),
            "total_complexity": total_complexity,
            "avg_complexity": total_complexity / len(entities) if entities else 0,
        }
