"""Parser utilities for CodeMap.

Provides helper functions for file parsing operations including:
- Language detection from file extensions
- Safe file reading with encoding detection
- Path normalization
- Line counting and metrics calculation

Example
-------
>>> from codemap.parsers.utils import detect_language, read_file_safe
>>> detect_language(Path("main.py"))
'python'
>>> content = read_file_safe(Path("example.js"))
"""

from pathlib import Path
from typing import Optional, Dict, Any
import re


def detect_language(file_path: Path) -> Optional[str]:
    """Detect programming language from file extension.

    Args:
        file_path: Path to the file to analyze.

    Returns:
        Language identifier string (e.g., 'python', 'javascript')
        or None if extension is not recognized.

    Examples:
        >>> detect_language(Path("main.py"))
        'python'
        >>> detect_language(Path("utils.js"))
        'javascript'
        >>> detect_language(Path("README"))
        None
    """
    ext_map = {
        # Python
        ".py": "python",
        ".pyi": "python",
        ".pyw": "python",
        ".pyx": "python",
        ".pxd": "python",
        ".pxi": "python",
        # JavaScript / TypeScript
        ".js": "javascript",
        ".jsx": "javascript",
        ".mjs": "javascript",
        ".cjs": "javascript",
        ".ts": "typescript",
        ".tsx": "typescript",
        ".mts": "typescript",
        ".cts": "typescript",
        # Java / JVM
        ".java": "java",
        ".kt": "kotlin",
        ".kts": "kotlin",
        ".scala": "scala",
        ".groovy": "groovy",
        ".jsp": "java",
        # C / C++
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
        # C#
        ".cs": "csharp",
        ".csx": "csharp",
        ".cake": "csharp",
        # Go
        ".go": "go",
        # Rust
        ".rs": "rust",
        ".rlib": "rust",
        # Ruby
        ".rb": "ruby",
        ".erb": "ruby",
        ".rhtml": "ruby",
        ".rake": "ruby",
        ".gemspec": "ruby",
        ".rbx": "ruby",
        ".rjs": "ruby",
        # PHP
        ".php": "php",
        ".phtml": "php",
        ".php3": "php",
        ".php4": "php",
        ".php5": "php",
        ".phar": "php",
        # Swift
        ".swift": "swift",
        ".swiftsource": "swift",
        # Objective-C
        ".m": "objective-c",
        ".mm": "objective-c",
        ".h": "objective-c",
        # Kotlin
        ".kt": "kotlin",
        ".kts": "kotlin",
        # Scala
        ".scala": "scala",
        # Groovy
        ".groovy": "groovy",
        ".gvy": "groovy",
        ".gy": "groovy",
        ".gsh": "groovy",
        # Shell / Script
        ".sh": "shell",
        ".bash": "shell",
        ".zsh": "shell",
        ".fish": "shell",
        ".ksh": "shell",
        ".ash": "shell",
        ".bashrc": "shell",
        ".sh": "shell",
        ".ps1": "powershell",
        ".psm1": "powershell",
        ".bat": "batch",
        ".cmd": "batch",
        # Perl
        ".pl": "perl",
        ".pm": "perl",
        ".t": "perl",
        ".pod": "perl",
        # Python Script
        ".py": "python",
        ".pyw": "python",
        ".py": "python",
        # Lua
        ".lua": "lua",
        ".luac": "lua",
        # R
        ".r": "r",
        ".R": "r",
        ".rdata": "r",
        ".rds": "r",
        ".rproj": "r",
        # Julia
        ".jl": "julia",
        # Dart
        ".dart": "dart",
        # Elixir
        ".ex": "elixir",
        ".exs": "elixir",
        ".eex": "elixir",
        ".leex": "elixir",
        # Erlang
        ".erl": "erlang",
        ".hrl": "erlang",
        ".xrl": "erlang",
        ".yrl": "erlang",
        ".beam": "erlang",
        # Haskell
        ".hs": "haskell",
        ".lhs": "haskell",
        ".hs-boot": "haskell",
        ".lhs-boot": "haskell",
        # Clojure
        ".clj": "clojure",
        ".cljs": "clojure",
        ".cljc": "clojure",
        ".edn": "clojure",
        # F#
        ".fs": "fsharp",
        ".fsi": "fsharp",
        ".fsx": "fsharp",
        ".fsscript": "fsharp",
        # OCaml
        ".ml": "ocaml",
        ".mli": "ocaml",
        ".mll": "ocaml",
        ".mly": "ocaml",
        # Rust
        ".rs": "rust",
        # Zig
        ".zig": "zig",
        # Nim
        ".nim": "nim",
        ".nims": "nim",
        # Crystal
        ".cr": "crystal",
        # V
        ".v": "v",
        ".vv": "v",
        # Odin
        ".odin": "odin",
        # Web
        ".html": "html",
        ".htm": "html",
        ".xhtml": "html",
        ".css": "css",
        ".scss": "scss",
        ".sass": "scss",
        ".less": "less",
        ".styl": "stylus",
        ".xml": "xml",
        ".svg": "xml",
        ".yaml": "yaml",
        ".yml": "yaml",
        ".json": "json",
        ".jsonc": "json",
        ".jsonl": "json",
        ".toml": "toml",
        ".ini": "ini",
        ".cfg": "ini",
        ".conf": "ini",
        ".env": "dotenv",
        ".dotenv": "dotenv",
        # Markdown
        ".md": "markdown",
        ".markdown": "markdown",
        ".mdown": "markdown",
        ".mkd": "markdown",
        ".mkdn": "markdown",
        ".rmd": "markdown",
        # SQL
        ".sql": "sql",
        ".sqlite": "sql",
        ".ddl": "sql",
        ".dml": "sql",
        # Dockerfile / Containers
        "Dockerfile": "dockerfile",
        ".dockerfile": "dockerfile",
        ".dockerignore": "dockerignore",
        # Configuration
        ".yaml": "yaml",
        ".yml": "yaml",
        ".json": "json",
        ".toml": "toml",
        ".ini": "ini",
        ".cfg": "ini",
        ".conf": "ini",
        ".properties": "properties",
        ".props": "properties",
        ".xml": "xml",
        ".xsd": "xml",
        ".xsl": "xml",
        ".xslt": "xml",
        # Template
        ".jinja": "jinja",
        ".jinja2": "jinja",
        ".j2": "jinja",
        ".html.j2": "jinja",
        ".htm.j2": "jinja",
        ".ejs": "ejs",
        ".pug": "pug",
        ".jade": "pug",
        ".handlebars": "handlebars",
        ".hbs": "handlebars",
        ".mustache": "mustache",
        # Protocol Buffers
        ".proto": "protobuf",
        # GraphQL
        ".graphql": "graphql",
        ".gql": "graphql",
        # Graphviz
        ".dot": "dot",
        ".gv": "dot",
        # Log
        ".log": "log",
        ".txt": "text",
        # Build / Make
        "Makefile": "makefile",
        "makefile": "makefile",
        "GNUmakefile": "makefile",
        ".mk": "makefile",
        "CMakeLists.txt": "cmake",
        "CMakeCache.txt": "cmake",
        ".cmake": "cmake",
        ".cmake.in": "cmake",
        # Jupyter
        ".ipynb": "jupyter",
        ".jpynb": "jupyter",
        # Notebook
        ".notebook": "jupyter",
        # Bazel
        "BUILD": "bazel",
        "BUILD.bazel": "bazel",
        "WORKSPACE": "bazel",
        ".bzl": "bazel",
        # Gradle
        "build.gradle": "gradle",
        "build.gradle.kts": "gradle",
        "settings.gradle": "gradle",
        "settings.gradle.kts": "gradle",
        # Protocol Buffers
        ".proto": "protobuf",
        # IDL
        ".idl": "idl",
        ".odl": "idl",
        # Forth
        ".fs": "forth",
        ".fth": "forth",
        ".forth": "forth",
        # AutoIt
        ".au3": "autoit",
        # AutoHotkey
        ".ahk": "autohotkey",
        ".ahkl": "autohotkey",
        # Visual Basic
        ".vb": "vb",
        ".vbs": "vbscript",
        ".bas": "vb",
        ".frm": "vb",
        ".cls": "vb",
        # Pascal
        ".pas": "pascal",
        ".pp": "pascal",
        ".dpr": "pascal",
        ".lpr": "pascal",
        # Fortran
        ".f": "fortran",
        ".f90": "fortran",
        ".f95": "fortran",
        ".f03": "fortran",
        ".f08": "fortran",
        ".for": "fortran",
        ".f77": "fortran",
        # COBOL
        ".cob": "cobol",
        ".cbl": "cobol",
        ".cpy": "cobol",
        # Lisp
        ".lisp": "lisp",
        ".lsp": "lisp",
        ".scm": "scheme",
        ".ss": "scheme",
        ".el": "elisp",
        ".emacs": "elisp",
        ".cl": "lisp",
        ".jl": "lisp",
        # Prolog
        ".pl": "prolog",
        ".pro": "prolog",
        ".p": "prolog",
        # ML
        ".ml": "ml",
        ".mli": "ml",
        ".sml": "ml",
        ".sig": "ml",
        ".fun": "ml",
        # Smalltalk
        ".st": "smalltalk",
        ".cs": "smalltalk",
        # Apex
        ".cls": "apex",
        ".trigger": "apex",
        ".apex": "apex",
        # Visualforce
        ".page": "visualforce",
        ".component": "visualforce",
        # Lightning
        ".auradoc": "aura",
        ".css": "aura",
        ".design": "aura",
        ".svg": "aura",
        ".js": "aura",
        # Solidity
        ".sol": "solidity",
        # Vyper
        ".vy": "vyper",
        # Move
        ".move": "move",
        # Sui
        ".move": "sui",
        # Cairo
        ".cairo": "cairo",
        ".scarb": "cairo",
        # Motoko
        ".mo": "motoko",
        # Ink
        ".ink": "ink",
        # Q#
        ".qs": "qsharp",
        # Julia
        ".jl": "julia",
        # Stan
        ".stan": "stan",
        # NumPy
        ".npy": "numpy",
        ".npz": "numpy",
        # Pandas
        ".pkl": "pickle",
        ".pickle": "pickle",
        ".parquet": "parquet",
        # Arrow
        ".arrow": "arrow",
        ".feather": "arrow",
        # CSV
        ".csv": "csv",
        ".tsv": "csv",
        # LaTeX
        ".tex": "latex",
        ".latex": "latex",
        ".ltx": "latex",
        ".bib": "bibtex",
        ".cls": "latex",
        ".sty": "latex",
        # AsciiDoc
        ".adoc": "asciidoc",
        ".ascii": "asciidoc",
        ".ad": "asciidoc",
        # reStructuredText
        ".rst": "rst",
        ".rest": "rst",
        ".restx": "rst",
        # Textile
        ".textile": "textile",
        # Pod
        ".pod": "pod",
        # Texinfo
        ".texi": "texinfo",
        ".texinfo": "texinfo",
        # NSIS
        ".nsi": "nsis",
        ".nsh": "nsis",
        # Inno Setup
        ".iss": "inno",
        # WiX
        ".wxs": "wix",
        ".wxi": "wix",
        ".wxl": "wix",
        # GraphQL
        ".graphql": "graphql",
        ".gql": "graphql",
        ".graphqls": "graphql",
        # OpenAPI
        ".openapi": "openapi",
        ".oas": "openapi",
        ".swagger": "openapi",
        ".yaml": "openapi",
        ".yml": "openapi",
        # AsyncAPI
        ".asyncapi": "asyncapi",
        ".yaml": "asyncapi",
        ".yml": "asyncapi",
        # SQL
        ".sql": "sql",
        ".ddl": "sql",
        ".dml": "sql",
        ".dcl": "sql",
        ".tcl": "sql",
        # PL/SQL
        ".pls": "plsql",
        ".plsql": "plsql",
        ".pck": "plsql",
        ".pkb": "plsql",
        ".pks": "plsql",
        ".pkg": "plsql",
        ".body": "plsql",
        ".trg": "plsql",
        ".vw": "plsql",
        ".fnc": "plsql",
        ".prc": "plsql",
        ".tab": "plsql",
        ".syn": "plsql",
        ".seq": "plsql",
        ".synonym": "plsql",
        # T-SQL
        ".sql": "tsql",
        ".tsql": "tsql",
        # SQLite
        ".sqlite": "sqlite",
        ".db": "sqlite",
        ".sqlite3": "sqlite",
        # NoSQL
        ".cql": "cql",
        ".cypher": "cypher",
        ".gremlin": "gremlin",
        ".groovy": "gremlin",
        # Markdown
        ".md": "markdown",
        ".markdown": "markdown",
        ".mdown": "markdown",
        ".mkd": "markdown",
        ".mkdn": "markdown",
        ".mdwn": "markdown",
        ".mdtxt": "markdown",
        ".mdtext": "markdown",
        ".rmd": "markdown",
        ".ron": "markdown",
        # Wiki
        ".wiki": "wiki",
        ".mediawiki": "wiki",
        ".mw": "wiki",
        # Config
        ".json": "json",
        ".json5": "json",
        ".jsonc": "json",
        ".jsonl": "json",
        ".ndjson": "json",
        ".yaml": "yaml",
        ".yml": "yaml",
        ".toml": "toml",
        ".ini": "ini",
        ".cfg": "ini",
        ".conf": "ini",
        ".properties": "properties",
        ".props": "properties",
        ".env": "dotenv",
        ".dotenv": "dotenv",
        ".rc": "rc",
        ".eslintrc": "json",
        ".prettierrc": "json",
        ".babelrc": "json",
        ".jscsrc": "json",
        ".jshintrc": "json",
        ".htmlhintrc": "json",
        ".stylelintrc": "json",
        ".editorconfig": "ini",
    }
    return ext_map.get(file_path.suffix.lower())


def get_file_encoding(file_path: Path) -> str:
    """Detect file encoding, defaulting to UTF-8.

    Uses chardet library if available for accurate encoding detection.

    Args:
        file_path: Path to the file to check.

    Returns:
        Detected encoding string (e.g., 'utf-8', 'latin-1').
        Defaults to 'utf-8' if chardet is not installed.
    """
    try:
        import chardet

        with open(file_path, "rb") as f:
            result = chardet.detect(f.read(1024 * 1024))
            return result.get("encoding", "utf-8")
    except ImportError:
        return "utf-8"


def read_file_safe(file_path: Path) -> Optional[str]:
    """Read file safely with encoding detection and error handling.

    Args:
        file_path: Path to the file to read.

    Returns:
        File contents as string, or None if reading fails.
    """
    try:
        encoding = get_file_encoding(file_path)
        with open(file_path, "r", encoding=encoding, errors="replace") as f:
            return f.read()
    except Exception:
        return None


def normalize_path(path: Path, base: Path) -> str:
    """Normalize path relative to base directory.

    Args:
        path: The absolute path to normalize.
        base: The base directory to make path relative to.

    Returns:
        Relative path string from base directory.

    Examples:
        >>> normalize_path(Path("/project/src/main.py"), Path("/project"))
        'src/main.py'
    """
    try:
        return str(path.relative_to(base))
    except ValueError:
        return str(path)


def extract_line_number(content: str, pattern: str) -> Optional[int]:
    """Extract line number where pattern first appears.

    Args:
        content: Full file content as string.
        pattern: String pattern to search for.

    Returns:
        Line number (1-indexed) where pattern first appears,
        or None if not found.
    """
    for i, line in enumerate(content.split("\n"), 1):
        if pattern in line:
            return i
    return None


def count_lines(content: str) -> int:
    """Count total lines in content including empty lines.

    Args:
        content: Text content to count lines in.

    Returns:
        Total number of lines.
    """
    return len(content.split("\n"))


def count_lines_of_code(content: str) -> int:
    """Count non-empty, non-comment lines of code.

    Handles single-line comments (//, #) and multi-line comments (/* */).

    Args:
        content: Source code content as string.

    Returns:
        Number of actual code lines (excluding comments and empty lines).

    Examples:
        >>> code = '''
        ... def hello():
        ...     # This is a comment
        ...     print("Hello")  # inline comment
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
