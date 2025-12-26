"""Gestor de Servicios de IA para CodeMap.

Administra múltiples proveedores de IA, proporciona failover automático,
caching de respuestas y una interfaz unificada para análisis de código.

Este módulo permite usar diferentes proveedores de IA (Ollama, OpenAI,
Anthropic) de manera transparente, cambiando automáticamente cuando
uno no está disponible.

Módulos
-------
providers.base
    Clases base y tipos comunes para proveedores de IA.
providers.ollama
    Proveedor para modelos locales de Ollama.
providers.openai
    Proveedor para la API de OpenAI.
providers.anthropic
    Proveedor para la API de Anthropic.

Ejemplo
-------
>>> from codemap.ai.service_manager import AIServiceManager
>>> manager = AIServiceManager()
>>> response = manager.generate_insights("def hello(): print('Hello')")
>>> print(response.text)
"""

from __future__ import annotations

import hashlib
import time
from pathlib import Path
from typing import Any, Dict, List, Optional, Type

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


class AIServiceManager:

    def __init__(
        self,
        preferred_provider: Optional[AIProviderType] = None,
        fallback_providers: Optional[List[AIProviderType]] = None,
        cache_enabled: bool = True,
        cache_ttl: int = 3600,
    ):
        """Inicializa el gestor de servicios de IA.

        Args:
            preferred_provider: Proveedor preferido (se usará primero).
            fallback_providers: Lista de proveedores de respaldo ordenados.
            cache_enabled: Si True, habilita el cache de respuestas.
            cache_ttl: Tiempo de vida del cache en segundos.
        """
        # Instance variables (not class variables!)
        self._providers: Dict[AIProviderType, BaseAIProvider] = {}
        self._provider_order: List[AIProviderType] = []
        self._cache: Dict[str, tuple[AIResponse, float]] = {}
        self._metrics: Dict[str, Dict[str, Any]] = {}
        self.cache_enabled = cache_enabled
        self.cache_ttl = cache_ttl

        # Configurar orden de proveedores
        if preferred_provider:
            self._provider_order = [preferred_provider]
        else:
            self._provider_order = [AIProviderType.OLLAMA]

        if fallback_providers:
            for provider in fallback_providers:
                if provider not in self._provider_order:
                    self._provider_order.append(provider)

        # Añadir proveedores cloud como fallback
        for provider_type in [AIProviderType.OPENAI, AIProviderType.ANTHROPIC]:
            if provider_type not in self._provider_order:
                self._provider_order.append(provider_type)

        # Inicializar métricas
        self._metrics = {
            pt.value: {
                "requests": 0,
                "successes": 0,
                "failures": 0,
                "total_latency_ms": 0.0,
                "total_tokens": 0,
            }
            for pt in AIProviderType
        }

    def register_provider(
        self, provider_type: AIProviderType, provider: BaseAIProvider
    ) -> None:
        """Registra un proveedor de IA.

        Args:
            provider_type: Tipo de proveedor a registrar.
            provider: Instancia del proveedor.
        """
        self._providers[provider_type] = provider
        if provider_type not in self._provider_order:
            self._provider_order.insert(0, provider_type)

    def unregister_provider(self, provider_type: AIProviderType) -> bool:
        """Elimina el registro de un proveedor.

        Args:
            provider_type: Tipo de proveedor a eliminar.

        Returns:
            True si se eliminó el proveedor.
        """
        if provider_type in self._providers:
            del self._providers[provider_type]
            return True
        return False

    def get_provider(self, provider_type: AIProviderType) -> Optional[BaseAIProvider]:
        """Obtiene un proveedor específico.

        Args:
            provider_type: Tipo de proveedor.

        Returns:
            Instancia del proveedor o None si no existe.
        """
        return self._providers.get(provider_type)

    def _get_best_provider(self) -> Optional[BaseAIProvider]:
        """Obtiene el mejor proveedor disponible.

        Returns:
            El proveedor con mayor prioridad que esté disponible.
        """
        for provider_type in self._provider_order:
            if provider_type in self._providers:
                provider = self._providers[provider_type]
                if provider.is_available():
                    return provider
        return None

    def _get_cache_key(self, action: str, code: str, **kwargs) -> str:
        """Genera una clave única para el cache.

        Args:
            action: Tipo de acción (insights, patterns, explain, refactor).
            code: Código fuente.
            kwargs: Argumentos adicionales.

        Returns:
            Hash único para la combinación de parámetros.
        """
        key_data = f"{action}:{code}:{sorted(kwargs.items())}"
        return hashlib.md5(key_data.encode()).hexdigest()

    def _get_cached_response(self, cache_key: str) -> Optional[AIResponse]:
        """Obtiene una respuesta del cache.

        Args:
            cache_key: Clave del cache.

        Returns:
            Respuesta en cache o None si no existe o expiró.
        """
        if not self.cache_enabled:
            return None

        if cache_key in self._cache:
            response, timestamp = self._cache[cache_key]
            if time.time() - timestamp < self.cache_ttl:
                return response
            else:
                del self._cache[cache_key]
        return None

    def _cache_response(self, cache_key: str, response: AIResponse) -> None:
        """Guarda una respuesta en el cache.

        Args:
            cache_key: Clave del cache.
            response: Respuesta a guardar.
        """
        if self.cache_enabled and response.success:
            self._cache[cache_key] = (response, time.time())

    def _update_metrics(
        self, provider_type: AIProviderType, response: AIResponse
    ) -> None:
        """Actualiza las métricas de uso.

        Args:
            provider_type: Tipo de proveedor utilizado.
            response: Respuesta received.
        """
        pt_value = provider_type.value
        m = self._metrics[pt_value]
        m["requests"] += 1
        m["total_latency_ms"] += response.latency_ms
        m["total_tokens"] += response.tokens_used
        if response.success:
            m["successes"] += 1
        else:
            m["failures"] += 1

    def _execute_with_fallback(
        self,
        action: str,
        code: str,
        provider_action: str,
        **kwargs,
    ) -> AIResponse:
        """Ejecuta una acción con fallback entre proveedores.

        Args:
            action: Nombre de la acción (para cache).
            code: Código fuente.
            provider_action: Método a llamar en el proveedor.
            kwargs: Argumentos adicionales.

        Returns:
            AIResponse del proveedor que respondió.
        """
        # Verificar cache
        cache_key = self._get_cache_key(action, code, **kwargs)
        cached = self._get_cached_response(cache_key)
        if cached:
            cached.metadata["cached"] = True
            return cached

        provider = self._get_best_provider()
        if not provider:
            return AIResponse(
                text="",
                provider=AIProviderType.OPENAI,
                model="",
                success=False,
                error="No hay proveedores de IA disponibles",
            )

        # Ejecutar la acción
        provider_method = getattr(provider, provider_action)
        response = provider_method(code, **kwargs)

        # Actualizar métricas
        self._update_metrics(provider.provider_type, response)

        # Cachear respuesta exitosa
        if response.success:
            self._cache_response(cache_key, response)

        return response

    def generate_insights(
        self, code: str, context: Optional[str] = None
    ) -> AIResponse:
        """Genera insights sobre el código.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional.

        Returns:
            AIResponse con los insights.
        """
        return self._execute_with_fallback(
            "insights", code, "generate_insights", context=context
        )

    def analyze_patterns(
        self, code: str, file_path: Optional[Path] = None
    ) -> List[PatternMatch]:
        """Analiza patrones en el código.

        Args:
            code: Código fuente a analizar.
            file_path: Ruta del archivo.

        Returns:
            Lista de patrones detectados.
        """
        cache_key = self._get_cache_key("patterns", code, file=str(file_path))
        cached = self._get_cached_response(cache_key)
        if cached:
            return []

        provider = self._get_best_provider()
        if not provider:
            return []

        response = provider.analyze_patterns(code, file_path)
        self._update_metrics(provider.provider_type, AIResponse(text="", provider=provider.provider_type, model=provider.model))

        return response

    def explain_code(
        self, code: str, focus_lines: Optional[List[int]] = None
    ) -> AIResponse:
        """Explica una sección de código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar.

        Returns:
            AIResponse con la explicación.
        """
        return self._execute_with_fallback(
            "explain", code, "explain_code", focus_lines=focus_lines
        )

    def suggest_refactoring(
        self, code: str, issue_type: str = "general"
    ) -> AIResponse:
        """Sugiere refactorizaciones para el código.

        Args:
            code: Código fuente.
            issue_type: Tipo de problema.

        Returns:
            AIResponse con sugerencias.
        """
        return self._execute_with_fallback(
            "refactor", code, "suggest_refactoring", issue_type=issue_type
        )

    def clear_cache(self) -> None:
        """Limpia el cache de respuestas."""
        self._cache.clear()

    def get_metrics(self) -> Dict[str, Dict[str, Any]]:
        """Obtiene las métricas de uso de los proveedores.

        Returns:
            Diccionario con métricas por proveedor.
        """
        # Calcular promedios
        for m in self._metrics.values():
            if m["requests"] > 0:
                m["avg_latency_ms"] = round(
                    m["total_latency_ms"] / m["requests"], 2
                )
                m["success_rate"] = round(
                    m["successes"] / m["requests"] * 100, 2
                )
            else:
                m["avg_latency_ms"] = 0.0
                m["success_rate"] = 0.0

        return self._metrics

    def get_status(self) -> Dict[str, Any]:
        """Obtiene el estado general del gestor.

        Returns:
            Diccionario con el estado de cada proveedor.
        """
        status = {
            "providers": {},
            "cache_size": len(self._cache),
            "cache_enabled": self.cache_enabled,
            "cache_ttl": self.cache_ttl,
        }

        for provider_type in AIProviderType:
            pt_value = provider_type.value
            if provider_type in self._providers:
                provider = self._providers[provider_type]
                status["providers"][pt_value] = {
                    "registered": True,
                    "available": provider.is_available(),
                    "model": provider.model,
                }
            else:
                status["providers"][pt_value] = {
                    "registered": False,
                    "available": False,
                    "model": None,
                }

        return status

    def setup_ollama(
        self,
        model: str = "llama2",
        host: str = "http://localhost:11434",
        temperature: float = 0.7,
    ) -> OllamaProvider:
        """Configura y registra el proveedor de Ollama.

        Args:
            model: Modelo a utilizar.
            host: Host del servidor Ollama.
            temperature: Temperatura para generación.

        Returns:
            Instancia del proveedor de Ollama.
        """
        provider = OllamaProvider(
            model=model,
            host=host,
            temperature=temperature,
        )
        self.register_provider(AIProviderType.OLLAMA, provider)
        return provider

    def setup_openai(
        self,
        api_key: str,
        model: str = "gpt-4",
        temperature: float = 0.7,
    ) -> OpenAIProvider:
        """Configura y registra el proveedor de OpenAI.

        Args:
            api_key: Clave API de OpenAI.
            model: Modelo a utilizar.
            temperature: Temperatura para generación.

        Returns:
            Instancia del proveedor de OpenAI.
        """
        provider = OpenAIProvider(
            api_key=api_key,
            model=model,
            temperature=temperature,
        )
        self.register_provider(AIProviderType.OPENAI, provider)
        return provider

    def setup_anthropic(
        self,
        api_key: str,
        model: str = "claude-sonnet-4-20250514",
        temperature: float = 0.7,
    ) -> AnthropicProvider:
        """Configura y registra el proveedor de Anthropic.

        Args:
            api_key: Clave API de Anthropic.
            model: Modelo a utilizar.
            temperature: Temperatura para generación.

        Returns:
            Instancia del proveedor de Anthropic.
        """
        provider = AnthropicProvider(
            api_key=api_key,
            model=model,
            temperature=temperature,
        )
        self.register_provider(AIProviderType.ANTHROPIC, provider)
        return provider

    def auto_setup(
        self,
        ollama_model: Optional[str] = None,
        openai_key: Optional[str] = None,
        anthropic_key: Optional[str] = None,
    ) -> Dict[str, bool]:
        """Configura automáticamente los proveedores disponibles.

        Args:
            ollama_model: Modelo de Ollama a usar (si está disponible).
            openai_key: Clave API de OpenAI.
            anthropic_key: Clave API de Anthropic.

        Returns:
            Diccionario con los proveedores configurados.
        """
        results = {}

        # Intentar configurar Ollama
        if ollama_model:
            ollama = self.setup_ollama(model=ollama_model)
            results["ollama"] = ollama.test_connection()
        else:
            ollama = self.setup_ollama()
            results["ollama"] = ollama.is_available()

        # Configurar OpenAI si hay clave
        if openai_key:
            openai = self.setup_openai(api_key=openai_key)
            results["openai"] = openai.test_connection()
        else:
            results["openai"] = False

        # Configurar Anthropic si hay clave
        if anthropic_key:
            anthropic = self.setup_anthropic(api_key=anthropic_key)
            results["anthropic"] = anthropic.test_connection()
        else:
            results["anthropic"] = False

        return results

    def __repr__(self) -> str:
        """Representación en string del gestor."""
        available = sum(
            1 for p in self._providers.values() if p.is_available()
        )
        return (
            f"AIServiceManager("
            f"providers={len(self._providers)}, "
            f"available={available}, "
            f"cache={len(self._cache)})"
        )

    def __str__(self) -> str:
        """String descriptivo del gestor."""
        status = self.get_status()
        lines = ["AIServiceManager Status:"]
        for name, info in status["providers"].items():
            state = "✓" if info["available"] else "✗"
            model = info["model"] or "N/A"
            lines.append(f"  {state} {name}: {model}")
        return "\n".join(lines)
