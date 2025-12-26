"""Proveedor Base de IA para CodeMap.

Define la interfaz abstracta y tipos comunes para todos los proveedores
de IA (Ollama, OpenAI, Anthropic). Proporciona funcionalidades comunes
para análisis de código, detección de patrones y generación de insights.

Módulos
-------
base
    Clases base y tipos de resultados para todos los proveedores de IA.
ollama
    Proveedor para modelos locales de Ollama.
openai
    Proveedor para la API de OpenAI (GPT-4, GPT-3.5).
anthropic
    Proveedor para la API de Anthropic (Claude).

Ejemplo
-------
>>> from codemap.ai.providers.base import BaseAIProvider, AIProviderType
>>> from codemap.ai.providers.ollama import OllamaProvider
>>> provider = OllamaProvider()
>>> response = provider.generate_insights("def foo():\n    pass")
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path
from typing import Any, Dict, List, Optional


class AIProviderType(Enum):
    """Tipos de proveedores de IA soportados.

    Atributos:
        OLLAMA: Modelo local usando Ollama.
        OPENAI: API de OpenAI (GPT-4, GPT-3.5).
        ANTHROPIC: API de Anthropic (Claude).
        CUSTOM_OPENAI: Servidor compatible con OpenAI personalizado.
        LOCAL_FALLBACK: Proveedor local como respaldo.
    """
    OLLAMA = "ollama"
    OPENAI = "openai"
    ANTHROPIC = "anthropic"
    CUSTOM_OPENAI = "custom_openai"
    LOCAL_FALLBACK = "local_fallback"


@dataclass
class AIResponse:
    """Respuesta de un proveedor de IA.

    Contiene el resultado generado por el modelo junto con metadatos
    sobre el procesamiento, tokens usados y métricas de rendimiento.

    Atributos:
        text: Texto generado por el modelo de IA.
        provider: Tipo de proveedor que generó la respuesta.
        model: Nombre del modelo utilizado.
        tokens_used: Número de tokens utilizados en la petición.
        latency_ms: Tiempo de respuesta en milisegundos.
        success: Indica si la petición fue exitosa.
        error: Mensaje de error si la petición falló.
        metadata: Información adicional del proveedor.
    """
    text: str
    provider: AIProviderType
    model: str
    tokens_used: int = 0
    latency_ms: float = 0.0
    success: bool = True
    error: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class CodeInsight:
    """Insight generado por IA sobre el código analizado.

    Representa un análisis o sugerencia realizada por el modelo de IA
    sobre una porción específica de código.

    Atributos:
        type: Tipo de insight (bug, refactor, performance, security, etc.)
        title: Título descriptivo del insight.
        description: Descripción detallada del insight.
        file: Archivo donde se encontró el insight.
        line: Número de línea donde comienza el issue.
        severity: Nivel de severidad (low, medium, high, critical).
        suggestion: Sugerencia de código para resolver el issue.
        confidence: Confianza del modelo en el insight (0-1).
    """
    type: str
    title: str
    description: str
    file: str
    line: int
    severity: str = "medium"
    suggestion: Optional[str] = None
    confidence: float = 0.0


@dataclass
class PatternMatch:
    """Patrón de código detectado por IA.

    Representa un patrón arquitectónico, anti-patrón u otra estructura
    de código identificada por el análisis de IA.

    Atributos:
        name: Nombre del patrón detectado.
        category: Categoría del patrón (architecture, design, code_smell).
        description: Descripción del patrón encontrado.
        file: Archivo donde se encontró el patrón.
        line: Línea donde comienza el patrón.
        matched_code: Fragmento de código que coincide con el patrón.
        suggestions: Sugerencias relacionadas con el patrón.
    """
    name: str
    category: str
    description: str
    file: str
    line: int
    matched_code: str = ""
    suggestions: List[str] = field(default_factory=list)


class BaseAIProvider(ABC):
    """Clase base abstracta para todos los proveedores de IA.

    Define la interfaz común que deben implementar todos los proveedores
    de modelos de lenguaje. Proporciona funcionalidades compartidas para
    configuración, manejo de errores y caching básico.

    Atributos:
        provider_type: Tipo de proveedor (OLLAMA, OPENAI, ANTHROPIC, etc.)
        model: Nombre del modelo utilizado.
        api_key: Clave API para proveedores cloud.
        base_url: URL base del servidor API.
        timeout: Timeout para peticiones en segundos.
        max_retries: Número máximo de reintentos en caso de error.

    Ejemplo
    -------
    >>> class MyProvider(BaseAIProvider):
    ...     def generate_insights(self, code: str) -> AIResponse:
    ...         # Implementación del proveedor
    ...         pass
    """

    provider_type: AIProviderType
    model: str = "default"
    api_key: Optional[str] = None
    base_url: Optional[str] = None
    timeout: int = 60
    max_retries: int = 3
    temperature: float = 0.7
    max_tokens: int = 2048

    def __init__(
        self,
        model: Optional[str] = None,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: int = 2048,
        timeout: int = 60,
        max_retries: int = 3,
    ):
        """Inicializa el proveedor de IA.

        Args:
            model: Nombre del modelo a utilizar.
            api_key: Clave API para autenticación.
            base_url: URL base del servidor API.
            temperature: Temperatura para generación (0-1).
            max_tokens: Máximo número de tokens en la respuesta.
            timeout: Timeout para peticiones en segundos.
            max_retries: Número máximo de reintentos.
        """
        if model:
            self.model = model
        if api_key:
            self.api_key = api_key
        if base_url:
            self.base_url = base_url
        self.temperature = temperature
        self.max_tokens = max_tokens
        self.timeout = timeout
        self.max_retries = max_retries

    @property
    def provider_name(self) -> str:
        """Nombre descriptivo del proveedor."""
        return f"{self.provider_type.value} ({self.model})"

    @abstractmethod
    def generate_insights(self, code: str, context: Optional[str] = None) -> AIResponse:
        """Genera insights sobre el código proporcionado.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional (nombre del archivo, propósito, etc.)

        Returns:
            AIResponse con los insights generados.
        """
        pass

    @abstractmethod
    def analyze_patterns(
        self, code: str, file_path: Optional[Path] = None
    ) -> List[PatternMatch]:
        """Analiza patrones en el código.

        Args:
            code: Código fuente a analizar.
            file_path: Ruta del archivo para contexto adicional.

        Returns:
            Lista de patrones detectados.
        """
        pass

    @abstractmethod
    def explain_code(self, code: str, focus_lines: Optional[List[int]] = None) -> AIResponse:
        """Explica una sección de código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar (opcional).

        Returns:
            AIResponse con la explicación.
        """
        pass

    @abstractmethod
    def suggest_refactoring(
        self, code: str, issue_type: str = "general"
    ) -> AIResponse:
        """Sugiere refactorizaciones para el código.

        Args:
            code: Código fuente a refactorizar.
            issue_type: Tipo de problema (complexity, duplication, etc.)

        Returns:
            AIResponse con sugerencias de refactorización.
        """
        pass

    @abstractmethod
    def is_available(self) -> bool:
        """Verifica si el proveedor está disponible.

        Returns:
            True si el proveedor puede atender peticiones.
        """
        pass

    @abstractmethod
    def test_connection(self) -> bool:
        """Prueba la conexión con el proveedor.

        Returns:
            True si la conexión fue exitosa.
        """
        pass

    def _call_api(
        self,
        messages: List[Dict[str, str]],
        max_tokens: Optional[int] = None,
        temperature: Optional[float] = None,
    ) -> AIResponse:
        """Método base para realizar llamadas a la API.

        Args:
            messages: Lista de mensajes en formato conversación.
            max_tokens: Máximo de tokens en la respuesta.
            temperature: Temperatura para generación.

        Returns:
            AIResponse con la respuesta del modelo.
        """
        raise NotImplementedError(
            f"{self.__class__.__name__} debe implementar _call_api"
        )

    def _handle_error(self, error: Exception) -> AIResponse:
        """Maneja errores de manera consistente.

        Args:
            error: Excepción capturada.

        Returns:
            AIResponse con el error formateado.
        """
        return AIResponse(
            text="",
            provider=self.provider_type,
            model=self.model,
            success=False,
            error=str(error),
        )

    def _format_code_for_prompt(
        self, code: str, max_lines: int = 100, max_chars: int = 5000
    ) -> str:
        """Formatea el código para incluirlo en un prompt.

        Args:
            code: Código fuente original.
            max_lines: Máximo número de líneas a incluir.
            max_chars: Máximo número de caracteres.

        Returns:
            Código formateado para el prompt.
        """
        lines = code.split("\n")
        if len(lines) > max_lines:
            lines = lines[:max_lines]
            code = "\n".join(lines) + "\n... (código truncado)"
        elif len(code) > max_chars:
            code = code[:max_chars] + "\n... (código truncado)"
        return code

    def __repr__(self) -> str:
        """Representación en string del proveedor."""
        return f"<{self.__class__.__name__}: {self.provider_name}>"

    def __str__(self) -> str:
        """String descriptivo del proveedor."""
        return f"Proveedor {self.provider_name}"