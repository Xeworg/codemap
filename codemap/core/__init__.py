"""Módulo Core de CodeMap.

Contiene la configuración central, excepciones personalizadas y sistema
de eventos para la comunicación entre componentes.

Módulos
-------
config
    Configuración global y gestión de settings.
exceptions
    Excepciones personalizadas de CodeMap.
events
    Sistema de eventos y signals para comunicación entre componentes.

Ejemplo
-------
>>> from codemap.core.config import Config
>>> config = Config.load()
>>> config.ai.provider
'ollama'
"""

from codemap.core.config import Config, AIConfig, UIConfig
from codemap.core.exceptions import (
    CodeMapError,
    AnalysisError,
    ConfigurationError,
    ProviderError,
)

__all__ = [
    "Config",
    "AIConfig",
    "UIConfig",
    "CodeMapError",
    "AnalysisError",
    "ConfigurationError",
    "ProviderError",
]
