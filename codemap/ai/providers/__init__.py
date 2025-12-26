"""Proveedores de IA para CodeMap.

Contiene las implementaciones de los diferentes proveedores de modelos
de lenguaje para análisis de código.

Módulos
-------
base
    Clases base abstractas y tipos comunes para todos los proveedores.
ollama
    Proveedor para modelos locales de Ollama.
openai
    Proveedor para la API de OpenAI (GPT-4, GPT-3.5).
anthropic
    Proveedor para la API de Anthropic (Claude).

Ejemplo
-------
>>> from codemap.ai.providers import OllamaProvider, OpenAIProvider
>>> ollama = OllamaProvider(model="llama2")
>>> openai = OpenAIProvider(api_key="sk-...")
"""

from codemap.ai.providers.base import (
    AIProviderType,
    AIResponse,
    BaseAIProvider,
    CodeInsight,
    PatternMatch,
)
from codemap.ai.providers.ollama import OllamaProvider
from codemap.ai.providers.openai import OpenAIProvider
from codemap.ai.providers.anthropic import AnthropicProvider

__all__ = [
    # Tipos base
    "AIProviderType",
    "AIResponse",
    "BaseAIProvider",
    "CodeInsight",
    "PatternMatch",
    # Proveedores
    "OllamaProvider",
    "OpenAIProvider",
    "AnthropicProvider",
]
