"""Utilidades de parser para CodeMap.

Proporciona funciones auxiliares para operaciones de análisis de archivos incluyendo:
- Detección de lenguaje desde extensiones de archivo
- Lectura segura de archivos con detección de codificación
- Normalización de rutas
- Conteo de líneas y cálculo de métricas

Ejemplo
-------
>>> from codemap.parsers.utils import detect_language, read_file_safe
>>> detect_language(Path("main.py"))
'python'
>>> content = read_file_safe(Path("ejemplo.js"))
"""

from pathlib import Path
from typing import Optional, Dict, Any
import re


def detect_language(file_path: Path) -> Optional[str]:
    """Detecta el lenguaje de programación desde la extensión del archivo.

    Args:
        file_path: Ruta al archivo a analizar.

    Returns:
        Identificador del lenguaje (ej: 'python', 'javascript')
        o None si la extensión no es reconocida.

    Ejemplos:
        >>> detect_language(Path("main.py"))
        'python'
        >>> detect_language(Path("utils.js"))
        'javascript'
        >>> detect_language(Path("README"))
        None
    """
    ext_map = {
        ".py": "python",
        ".pyi": "python",
        ".pyw": "python",
        ".pyx": "python",
        ".pxd": "python",
        ".pxi": "python",
        ".js": "javascript",
        ".jsx": "javascript",
        ".mjs": "javascript",
        ".cjs": "javascript",
        ".ts": "typescript",
        ".tsx": "typescript",
        ".mts": "typescript",
        ".cts": "typescript",
        ".java": "java",
        ".kt": "kotlin",
        ".kts": "kotlin",
        ".scala": "scala",
        ".groovy": "groovy",
        ".jsp": "java",
        ".c": "c",
        ".h": "c",
        ".cpp": "cpp",
        ".cc": "cpp",
        ".cxx": "cpp",
        ".hpp": "cpp",
        ".hxx": "cpp",
        ".c++": "cpp",
        ".h++": "cpp",
        ".ipp": "cpp",
        ".inl": "cpp",
        ".ixx": "cpp",
        ".tpp": "cpp",
        ".txx": "cpp",
        ".cs": "csharp",
        ".csx": "csharp",
        ".cake": "csharp",
        ".go": "go",
        ".rs": "rust",
        ".rlib": "rust",
        ".rb": "ruby",
        ".erb": "ruby",
        ".rhtml": "ruby",
        ".rake": "ruby",
        ".gemspec": "ruby",
        ".rbx": "ruby",
        ".rjs": "ruby",
        ".php": "php",
        ".phtml": "php",
        ".php3": "php",
        ".php4": "php",
        ".php5": "php",
        ".phar": "php",
        ".swift": "swift",
        ".swiftsource": "swift",
        ".m": "objective-c",
        ".mm": "objective-c",
        ".h": "objective-c",
    }
    return ext_map.get(file_path.suffix.lower())


def get_file_encoding(file_path: Path) -> str:
    """Detecta la codificación del archivo, usando UTF-8 por defecto.

    Usa la biblioteca chardet si está disponible para una detección precisa.

    Args:
        file_path: Ruta al archivo a verificar.

    Returns:
        String de codificación detectada (ej: 'utf-8', 'latin-1').
        Retorna 'utf-8' como fallback.
    """
    try:
        import chardet

        with open(file_path, "rb") as f:
            result = chardet.detect(f.read(1024 * 1024))
            encoding = result.get("encoding")
            return encoding if encoding else "utf-8"
    except ImportError:
        return "utf-8"


def read_file_safe(file_path: Path) -> Optional[str]:
    """Lee un archivo de forma segura con detección de manejo de errores.

    Args:
        file_path: Ruta al archivo a leer.

    Returns:
        Contenido del archivo como string, o None si falla la lectura.
    """
    try:
        encoding = get_file_encoding(file_path)
        with open(file_path, "r", encoding=encoding, errors="replace") as f:
            return f.read()
    except Exception:
        return None


def normalize_path(path: Path, base: Path) -> str:
    """Normaliza la ruta relativa al directorio base.

    Args:
        path: La ruta absoluta a normalizar.
        base: El directorio base para hacer la ruta relativa.

    Returns:
        String de ruta relativa desde el directorio base.

    Ejemplos:
        >>> normalize_path(Path("/project/src/main.py"), Path("/project"))
        'src/main.py'
    """
    try:
        return str(path.relative_to(base))
    except ValueError:
        return str(path)


def extract_line_number(content: str, pattern: str) -> Optional[int]:
    """Extrae el número de línea donde aparece el patrón por primera vez.

    Args:
        content: Contenido completo del archivo como string.
        pattern: Patrón de string a buscar.

    Returns:
        Número de línea (1-indexed) donde aparece el patrón por primera vez,
        o None si no se encuentra.
    """
    for i, line in enumerate(content.split("\n"), 1):
        if pattern in line:
            return i
    return None


def count_lines(content: str) -> int:
    """Cuenta el total de líneas incluyendo líneas vacías.

    Args:
        content: Contenido de texto donde contar líneas.

    Returns:
        Número total de líneas.
    """
    return len(content.split("\n"))


def count_lines_of_code(content: str) -> int:
    """Cuenta las líneas de código que no están vacías ni son comentarios.

    Maneja comentarios de una sola línea (//, #) y comentarios
    multi-línea (/* */).

    Args:
        content: Contenido del código fuente como string.

    Returns:
        Número de líneas de código real (excluyendo comentarios y líneas vacías).

    Ejemplos:
        >>> code = '''
        ... def hello():
        ...     # Este es un comentario
        ...     print("Hello")  # comentario inline
        ... '''
        >>> count_lines_of_code(code)
        2
    """
    lines = content.split("\n")
    code_lines = 0
    in_multiline_comment = False

    for line in lines:
        stripped = line.strip()

        if in_multiline_comment:
            if "*/" in stripped:
                in_multiline_comment = False
            continue

        if stripped.startswith("/*"):
            if "*/" not in stripped:
                in_multiline_comment = True
            continue

        if stripped.startswith("//") or stripped.startswith("#"):
            continue

        if stripped:
            code_lines += 1

    return code_lines
