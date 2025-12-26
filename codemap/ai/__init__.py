"""Módulo de Inteligencia Artificial para CodeMap.

Proporciona capacidades de IA para análisis de código, detección de patrones,
generación de insights y sugerencias de refactorización usando múltiples
proveedores de modelos de lenguaje (Ollama, OpenAI, Anthropic).

Módulos
-------
providers
    Proveedores de IA (Ollama, OpenAI, Anthropic).
service_manager
    Gestor centralizado de servicios de IA.

Ejemplo
-------
>>> from codemap.ai import AIServiceManager
>>> manager = AIServiceManager()
>>> manager.auto_setup(ollama_model="llama2")
>>> response = manager.generate_insights("def hello(): print('Hello')")
"""

from typing import Optional

from codemap.ai.service_manager import AIServiceManager
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
    # Gestor
    "AIServiceManager",
]


def create_service_manager(
    preferred: str = "ollama",
    openai_key: Optional[str] = None,
    anthropic_key: Optional[str] = None,
    ollama_model: Optional[str] = None,
) -> AIServiceManager:
    """Crea un gestor de servicios de IA configurado.

    Args:
        preferred: Proveedor preferido (ollama, openai, anthropic).
        openai_key: Clave API de OpenAI (opcional).
        anthropic_key: Clave API de Anthropic (opcional).
        ollama_model: Modelo de Ollama (opcional).

    Returns:
        Instancia de AIServiceManager configurada.
    """
    manager = AIServiceManager(
        preferred_provider=AIProviderType(preferred)
    )

    # Configurar proveedores
    if preferred == "ollama" or ollama_model:
        manager.setup_ollama(model=ollama_model or "llama2")

    if openai_key:
        manager.setup_openai(api_key=openai_key)

    if anthropic_key:
        manager.setup_anthropic(api_key=anthropic_key)

    return manager
