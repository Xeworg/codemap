"""Configuración de CodeMap.

Gestiona la configuración global de la aplicación, incluyendo opciones de IA,
interfaz de usuario y preferencias de análisis. Soporta carga desde archivo JSON
y detección automática de entorno.

Ejemplo
-------
>>> from codemap.core.config import Config
>>> config = Config.load()
>>> config.ai.provider
'ollama'
>>> config.ui.theme
'dark'
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, Optional


@dataclass
class AIConfig:
    """Configuración del motor de IA.

    Atributos:
        provider: Proveedor de IA (ollama, openai, anthropic, none).
        ollama_url: URL del servidor Ollama.
        ollama_model: Modelo de Ollama a usar.
        openai_key: Clave API de OpenAI.
        openai_model: Modelo de OpenAI a usar.
        anthropic_key: Clave API de Anthropic.
        anthropic_model: Modelo de Anthropic a usar.
        temperature: Temperatura para generación (0-1).
        max_tokens: Máximo de tokens en respuestas.
        cache_enabled: Habilitar cache de respuestas.
        cache_ttl: Tiempo de vida del cache en segundos.
    """
    provider: str = "ollama"
    ollama_url: str = "http://localhost:11434"
    ollama_model: str = "llama2"
    openai_key: str = ""
    openai_model: str = "gpt-4"
    anthropic_key: str = ""
    anthropic_model: str = "claude-sonnet-4-20250514"
    temperature: float = 0.7
    max_tokens: int = 4096
    cache_enabled: bool = True
    cache_ttl: int = 3600

    def to_dict(self) -> Dict[str, Any]:
        """Convierte la configuración a diccionario."""
        return {
            "provider": self.provider,
            "ollama_url": self.ollama_url,
            "ollama_model": self.ollama_model,
            "openai_key": self.openai_key,
            "openai_model": self.openai_model,
            "anthropic_key": self.anthropic_key,
            "anthropic_model": self.anthropic_model,
            "temperature": self.temperature,
            "max_tokens": self.max_tokens,
            "cache_enabled": self.cache_enabled,
            "cache_ttl": self.cache_ttl,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> AIConfig:
        """Crea AIConfig desde diccionario."""
        return cls(
            provider=data.get("provider", "ollama"),
            ollama_url=data.get("ollama_url", "http://localhost:11434"),
            ollama_model=data.get("ollama_model", "llama2"),
            openai_key=data.get("openai_key", ""),
            openai_model=data.get("openai_model", "gpt-4"),
            anthropic_key=data.get("anthropic_key", ""),
            anthropic_model=data.get("anthropic_model", "claude-sonnet-4-20250514"),
            temperature=data.get("temperature", 0.7),
            max_tokens=data.get("max_tokens", 4096),
            cache_enabled=data.get("cache_enabled", True),
            cache_ttl=data.get("cache_ttl", 3600),
        )


@dataclass
class UIConfig:
    """Configuración de la interfaz de usuario.

    Atributos:
        theme: Tema visual (dark, light, system).
        language: Idioma de la interfaz.
        window_width: Ancho de ventana por defecto.
        window_height: Alto de ventana por defecto.
        auto_analyze: Autoanalizar proyectos al abrir.
        show_line_numbers: Mostrar números de línea.
        node_layout: Algoritmo de disposición de nodos.
    """
    theme: str = "dark"
    language: str = "es"
    window_width: int = 1400
    window_height: int = 900
    auto_analyze: bool = True
    show_line_numbers: bool = True
    node_layout: str = "graphviz"

    def to_dict(self) -> Dict[str, Any]:
        """Convierte la configuración a diccionario."""
        return {
            "theme": self.theme,
            "language": self.language,
            "window_width": self.window_width,
            "window_height": self.window_height,
            "auto_analyze": self.auto_analyze,
            "show_line_numbers": self.show_line_numbers,
            "node_layout": self.node_layout,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> UIConfig:
        """Crea UIConfig desde diccionario."""
        return cls(
            theme=data.get("theme", "dark"),
            language=data.get("language", "es"),
            window_width=data.get("window_width", 1400),
            window_height=data.get("window_height", 900),
            auto_analyze=data.get("auto_analyze", True),
            show_line_numbers=data.get("show_line_numbers", True),
            node_layout=data.get("node_layout", "graphviz"),
        )


@dataclass
class AnalysisConfig:
    """Configuración del análisis de código.

    Atributos:
        supported_extensions: Extensiones de archivos a analizar.
        excluded_patterns: Patrones de archivos a excluir.
        max_file_size: Tamaño máximo de archivo en bytes.
        max_depth: Profundidad máxima de análisis.
        parallel_files: Número de archivos a procesar en paralelo.
    """
    supported_extensions: list = field(default_factory=lambda: [
        ".py", ".pyi", ".js", ".ts", ".java", ".jsx", ".tsx"
    ])
    excluded_patterns: list = field(default_factory=lambda: [
        "**/node_modules/**",
        "**/.git/**",
        "**/__pycache__/**",
        "**/venv/**",
        "**/.env/**",
        "**/*.min.js",
        "**/*.min.css",
    ])
    max_file_size: int = 5 * 1024 * 1024  # 5MB
    max_depth: int = 10
    parallel_files: int = 4

    def to_dict(self) -> Dict[str, Any]:
        """Convierte la configuración a diccionario."""
        return {
            "supported_extensions": self.supported_extensions,
            "excluded_patterns": self.excluded_patterns,
            "max_file_size": self.max_file_size,
            "max_depth": self.max_depth,
            "parallel_files": self.parallel_files,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> AnalysisConfig:
        """Crea AnalysisConfig desde diccionario."""
        return cls(
            supported_extensions=data.get("supported_extensions", [
                ".py", ".pyi", ".js", ".ts", ".java", ".jsx", ".tsx"
            ]),
            excluded_patterns=data.get("excluded_patterns", [
                "**/node_modules/**",
                "**/.git/**",
                "**/__pycache__/**",
                "**/venv/**",
                "**/.env/**",
            ]),
            max_file_size=data.get("max_file_size", 5 * 1024 * 1024),
            max_depth=data.get("max_depth", 10),
            parallel_files=data.get("parallel_files", 4),
        )


@dataclass
class Config:
    """Configuración global de CodeMap.

    Clase principal que contiene toda la configuración de la aplicación.
    Soporte para persistencia en archivo JSON y detección automática de entorno.

    Atributos:
        version: Versión de la configuración.
        ai: Configuración del motor de IA.
        ui: Configuración de la interfaz de usuario.
        analysis: Configuración del análisis.
        recent_projects: Proyectos recientes abiertos.
    """
    version: str = "1.0.0"
    ai: AIConfig = field(default_factory=AIConfig)
    ui: UIConfig = field(default_factory=UIConfig)
    analysis: AnalysisConfig = field(default_factory=AnalysisConfig)
    recent_projects: list = field(default_factory=list)

    CONFIG_FILENAME = "codemap.json"

    @classmethod
    def get_config_dir(cls) -> Path:
        """Obtiene el directorio de configuración.

        Returns:
            Path al directorio de configuración (~/.config/codemap).
        """
        config_dir = Path(os.environ.get(
            "XDG_CONFIG_HOME",
            Path.home() / ".config"
        )) / "codemap"
        config_dir.mkdir(parents=True, exist_ok=True)
        return config_dir

    @classmethod
    def get_config_path(cls) -> Path:
        """Obtiene la ruta del archivo de configuración.

        Returns:
            Path al archivo de configuración.
        """
        return cls.get_config_dir() / cls.CONFIG_FILENAME

    @classmethod
    def load(cls, config_path: Optional[Path] = None) -> Config:
        """Carga la configuración desde archivo JSON.

        Args:
            config_path: Ruta opcional al archivo de configuración.

        Returns:
            Instancia de Config con los valores cargados.
        """
        path = config_path or cls.get_config_path()

        if not path.exists():
            return cls()

        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
            return cls.from_dict(data)
        except (json.JSONDecodeError, OSError):
            return cls()

    def save(self, config_path: Optional[Path] = None) -> bool:
        """Guarda la configuración en archivo JSON.

        Args:
            config_path: Ruta opcional al archivo de configuración.

        Returns:
            True si se guardó exitosamente.
        """
        path = config_path or self.get_config_path()

        try:
            with open(path, "w", encoding="utf-8") as f:
                json.dump(self.to_dict(), f, indent=2, ensure_ascii=False)
            return True
        except OSError:
            return False

    def to_dict(self) -> Dict[str, Any]:
        """Convierte la configuración a diccionario."""
        return {
            "version": self.version,
            "ai": self.ai.to_dict(),
            "ui": self.ui.to_dict(),
            "analysis": self.analysis.to_dict(),
            "recent_projects": self.recent_projects,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> Config:
        """Crea Config desde diccionario."""
        return cls(
            version=data.get("version", "1.0.0"),
            ai=AIConfig.from_dict(data.get("ai", {})),
            ui=UIConfig.from_dict(data.get("ui", {})),
            analysis=AnalysisConfig.from_dict(data.get("analysis", {})),
            recent_projects=data.get("recent_projects", []),
        )

    def add_recent_project(self, path: Path) -> None:
        """Añade un proyecto a la lista de recientes.

        Args:
            path: Ruta del proyecto a añadir.
        """
        path_str = str(path.absolute())
        self.recent_projects = [
            p for p in self.recent_projects if p != path_str
        ]
        self.recent_projects.insert(0, path_str)
        self.recent_projects = self.recent_projects[:10]

    def remove_recent_project(self, path: Path) -> None:
        """Elimina un proyecto de la lista de recientes.

        Args:
            path: Ruta del proyecto a eliminar.
        """
        path_str = str(path.absolute())
        self.recent_projects = [p for p in self.recent_projects if p != path_str]

    def get_provider_config(self, provider: str) -> Dict[str, Any]:
        """Obtiene la configuración para un proveedor específico.

        Args:
            provider: Nombre del proveedor.

        Returns:
            Diccionario con la configuración del proveedor.
        """
        if provider == "ollama":
            return {
                "base_url": self.ai.ollama_url,
                "model": self.ai.ollama_model,
            }
        elif provider == "openai":
            return {
                "api_key": self.ai.openai_key,
                "model": self.ai.openai_model,
            }
        elif provider == "anthropic":
            return {
                "api_key": self.ai.anthropic_key,
                "model": self.ai.anthropic_model,
            }
        return {}
