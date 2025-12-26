"""Proveedor Anthropic para CodeMap.

Implementación del proveedor de IA para la API de Anthropic (Claude).
Soporta modelos Claude 3 (Haiku, Sonnet, Opus) con capacidades avanzadas
de razonamiento y análisis de código.

Ejemplo
-------
>>> from codemap.ai.providers.anthropic import AnthropicProvider
>>> provider = AnthropicProvider(api_key="sk-ant-...")
>>> response = provider.generate_insights("def calculate():\\n    return 42")
>>> print(response.text)
"""

from __future__ import annotations

import json
import re
import time
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import httpx

from codemap.ai.providers.base import (
    AIProviderType,
    AIResponse,
    BaseAIProvider,
    CodeInsight,
    PatternMatch,
)


class AnthropicProvider(BaseAIProvider):
    """Proveedor de IA para la API de Anthropic (Claude).

    Gestiona la comunicación con los modelos Claude de Anthropic.
    Ofrece excelente capacidad de razonamiento y análisis de código.

    Atributos:
        api_key: Clave API de Anthropic.
        base_url: URL base de la API de Anthropic.
        version: Versión de la API de Anthropic.

    Ejemplo
    -------
    >>> provider = AnthropicProvider(
    ...     model="claude-sonnet-4-20250514",
    ...     api_key="sk-ant-...",
    ... )
    """

    provider_type = AIProviderType.ANTHROPIC
    DEFAULT_MODEL = "claude-sonnet-4-20250514"
    DEFAULT_BASE_URL = "https://api.anthropic.com"
    API_VERSION = "2023-06-01"

    def __init__(
        self,
        model: Optional[str] = None,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
        timeout: int = 120,
        max_retries: int = 3,
    ):
        """Inicializa el proveedor de Anthropic.

        Args:
            model: Nombre del modelo (claude-haiku, claude-sonnet, claude-opus).
            api_key: Clave API de Anthropic.
            base_url: URL base personalizada.
            temperature: Temperatura para generación (0-1).
            max_tokens: Máximo de tokens en la respuesta.
            timeout: Timeout para peticiones en segundos.
            max_retries: Número máximo de reintentos.
        """
        super().__init__(
            model=model or self.DEFAULT_MODEL,
            api_key=api_key,
            base_url=base_url or self.DEFAULT_BASE_URL,
            temperature=temperature,
            max_tokens=max_tokens,
            timeout=timeout,
            max_retries=max_retries,
        )
        self._client: Optional[httpx.Client] = None

    @property
    def available_models(self) -> List[str]:
        """Modelos disponibles en Anthropic."""
        return [
            "claude-haiku-3-20250514",
            "claude-sonnet-4-20250514",
            "claude-opus-4-20250514",
            "claude-3-5-haiku-20241022",
            "claude-3-5-sonnet-20241022",
            "claude-3-opus-20240229",
            "claude-3-sonnet-20240229",
            "claude-3-haiku-20240307",
        ]

    def _get_client(self) -> httpx.Client:
        """Obtiene o crea el cliente HTTP."""
        if self._client is None:
            self._client = httpx.Client(
                timeout=self.timeout,
                headers={
                    "x-api-key": self.api_key or "",
                    "anthropic-version": self.API_VERSION,
                    "content-type": "application/json",
                },
            )
        return self._client

    def _close_client(self) -> None:
        """Cierra el cliente HTTP."""
        if self._client:
            self._client.close()
            self._client = None

    def _make_request(
        self, endpoint: str, data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Realiza una petición a la API de Anthropic.

        Args:
            endpoint: Endpoint de la API.
            data: Datos a enviar en formato JSON.

        Returns:
            Respuesta del servidor como diccionario.

        Raises:
            ValueError: Si la API key no está configurada.
            ConnectionError: Si hay error de conexión.
        """
        if not self.api_key:
            raise ValueError("API key de Anthropic no configurada")

        client = self._get_client()
        url = f"{self.base_url}/{endpoint.lstrip('/')}"

        response = client.post(url, json=data)

        if response.status_code == 200:
            return response.json()
        elif response.status_code == 401:
            raise ValueError("API key de Anthropic inválida")
        elif response.status_code == 429:
            raise ConnectionError("Límite de tasa excedido en Anthropic")
        else:
            raise ConnectionError(
                f"Error de Anthropic: {response.status_code} - {response.text}"
            )

    def _call_api(
        self,
        messages: List[Dict[str, str]],
        max_tokens: Optional[int] = None,
        temperature: Optional[float] = None,
    ) -> AIResponse:
        """Implementación de llamada a la API de Anthropic.

        Args:
            messages: Lista de mensajes en formato chat.
            max_tokens: Máximo de tokens en la respuesta.
            temperature: Temperatura para generación.

        Returns:
            AIResponse con la respuesta del modelo.
        """
        start_time = time.perf_counter()

        # Anthropic usa formato diferente: system message separado
        system_prompt = None
        anthropic_messages = []

        for msg in messages:
            if msg["role"] == "system":
                system_prompt = msg["content"]
            else:
                anthropic_messages.append(msg)

        data: Dict[str, Any] = {
            "model": self.model,
            "messages": anthropic_messages,
            "temperature": temperature or self.temperature,
            "max_tokens": max_tokens or self.max_tokens,
        }

        if system_prompt:
            data["system"] = system_prompt

        try:
            result = self._make_request("v1/messages", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            text = result["content"][0]["text"]

            usage = result.get("usage", {})
            tokens_used = usage.get("input_tokens", 0) + usage.get("output_tokens", 0)

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                tokens_used=tokens_used,
                latency_ms=latency_ms,
                success=True,
                metadata={
                    "stop_reason": result.get("stop_reason"),
                    "input_tokens": usage.get("input_tokens"),
                    "output_tokens": usage.get("output_tokens"),
                },
            )

        except Exception as e:
            latency_ms = (time.perf_counter() - start_time) * 1000
            return AIResponse(
                text="",
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=False,
                error=str(e),
            )

    def _build_insights_prompt(
        self, code: str, context: Optional[str] = None
    ) -> Tuple[str, str]:
        """Construye el prompt para generar insights.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional.

        Returns:
            Tupla con (system_prompt, user_prompt).
        """
        context_section = f"\n\nContexto: {context}" if context else ""

        user_content = f"""Analiza el siguiente código fuente y proporciona insights detallados.{context_section}

```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada insight identificado:
- **Tipo**: Título breve
  Descripción detallada
  Línea(s): N

Categorías a buscar:
- Bugs y errores potenciales
- Problemas de rendimiento
- Code smells y anti-patrones
- Mejores prácticas
- Seguridad
- Oportunidades de refactorización

Responde solo con los insights:"""

        system_prompt = """Eres un analizador de código experto. Tu análisis es riguroso, preciso y orientada a la acción. Siempre proporcionas insights específicos con números de línea cuando es posible."""

        return system_prompt, user_content

    def generate_insights(self, code: str, context: Optional[str] = None) -> AIResponse:
        """Genera insights sobre el código proporcionado.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional (nombre del archivo, propósito, etc.)

        Returns:
            AIResponse con los insights generados.
        """
        system_prompt, user_prompt = self._build_insights_prompt(code, context)

        messages = [
            {"role": "user", "content": user_prompt},
        ]

        # Claude usa el system prompt separado
        data: Dict[str, Any] = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": self.max_tokens,
            "system": system_prompt,
        }

        start_time = time.perf_counter()

        try:
            result = self._make_request("v1/messages", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            text = result["content"][0]["text"]
            usage = result.get("usage", {})
            tokens_used = usage.get("input_tokens", 0) + usage.get("output_tokens", 0)

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                tokens_used=tokens_used,
                latency_ms=latency_ms,
                success=True,
            )

        except Exception as e:
            latency_ms = (time.perf_counter() - start_time) * 1000
            return AIResponse(
                text="",
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=False,
                error=str(e),
            )

    def _build_patterns_prompt(
        self, code: str, file_path: Optional[Path] = None
    ) -> tuple:
        """Construye el prompt para detectar patrones.

        Args:
            code: Código fuente a analizar.
            file_path: Ruta del archivo para contexto.

        Returns:
            Tuple con (system_prompt, user_prompt).
        """
        file_context = f"\nArchivo: {file_path}" if file_path else ""

        system_prompt = """Eres un experto en patrones de diseño y arquitectura de software. Identificas patrones con precisión y los categorizas correctamente."""

        user_prompt = f"""Identifica patrones arquitectónicos y anti-patrones en el siguiente código.{file_context}

```code
{self._format_code_for_prompt(code, max_lines=200, max_chars=10000)}
```

Responde en formato JSON:
```json
{{
    "patterns": [
        {{
            "name": "nombre_del_patrón",
            "category": "architecture|design|code_smell|concurrency|testing",
            "description": "Descripción detallada",
            "line": número_de_línea,
            "matched_code": "fragmento",
            "suggestions": ["sugerencia1", "sugerencia2"]
        }}
    ]
}}
```

Responde únicamente con el JSON:"""

        return system_prompt, user_prompt

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
        system_prompt, user_prompt = self._build_patterns_prompt(code, file_path)

        messages = [{"role": "user", "content": user_prompt}]

        data = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": 1500,
            "system": system_prompt,
        }

        try:
            result = self._make_request("v1/messages", data)
            response_text = result["content"][0]["text"]

            json_match = re.search(r"\{[\s\S]*\}", response_text)
            if json_match:
                data = json.loads(json_match.group())
                patterns = []
                for p in data.get("patterns", []):
                    patterns.append(
                        PatternMatch(
                            name=p.get("name", "Unknown"),
                            category=p.get("category", "unknown"),
                            description=p.get("description", ""),
                            file=str(file_path) if file_path else "",
                            line=p.get("line", 0),
                            matched_code=p.get("matched_code", ""),
                            suggestions=p.get("suggestions", []),
                        )
                    )
                return patterns
            return []
        except Exception:
            return []

    def _build_explain_prompt(
        self, code: str, focus_lines: Optional[List[int]] = None
    ) -> tuple:
        """Construye el prompt para explicar código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar.

        Returns:
            Tuple con (system_prompt, user_prompt).
        """
        lines_guidance = (
            f"\n\nEnfócate especialmente en las líneas: {focus_lines}"
            if focus_lines
            else ""
        )

        system_prompt = """Eres un mentor de programación excepcional. Tus explicaciones son claras, didácticas y ayudarte a entender conceptos profundos."""

        user_prompt = f"""Explica el siguiente código de manera educativa.{lines_guidance}

```code
{self._format_code_for_prompt(code, max_lines=100)}
```

Incluye:
1. Propósito general
2. Explicación de secciones importantes
3. Conceptos clave
4. Ejemplo de uso

Sé claro y didáctico:"""

        return system_prompt, user_prompt

    def explain_code(self, code: str, focus_lines: Optional[List[int]] = None) -> AIResponse:
        """Explica una sección de código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar (opcional).

        Returns:
            AIResponse con la explicación.
        """
        system_prompt, user_prompt = self._build_explain_prompt(code, focus_lines)

        messages = [{"role": "user", "content": user_prompt}]

        data = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": 1500,
            "system": system_prompt,
        }

        start_time = time.perf_counter()

        try:
            result = self._make_request("v1/messages", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            text = result["content"][0]["text"]

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=True,
            )

        except Exception as e:
            latency_ms = (time.perf_counter() - start_time) * 1000
            return AIResponse(
                text="",
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=False,
                error=str(e),
            )

    def _build_refactor_prompt(
        self, code: str, issue_type: str = "general"
    ) -> tuple:
        """Construye el prompt para sugerir refactorizaciones.

        Args:
            code: Código fuente a refactorizar.
            issue_type: Tipo de problema.

        Returns:
            Tuple con (system_prompt, user_prompt).
        """
        issue_guidance = {
            "complexity": "Identifica funciones con alta complejidad y sugiere simplificaciones.",
            "duplication": "Identifica código duplicado y sugiere reutilización.",
            "naming": "Identifica nombres poco descriptivos.",
            "architecture": "Identifica problemas de diseño.",
            "general": "Identifica oportunidades de mejora.",
        }

        guidance = issue_guidance.get(issue_type, issue_guidance["general"])

        system_prompt = """Eres un experto en refactorización de código. Proporcionas sugerencias prácticas, detalladas y con código de ejemplo."""

        user_prompt = f"""Analiza y sugiere refactorizaciones para el código.{guidance}

```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada sugerencia:
1. Describe el problema
2. Explica por qué importa
3. Muestra el código refactorizado
4. Explica los beneficios

Estructura tu respuesta claramente:"""

        return system_prompt, user_prompt

    def suggest_refactoring(
        self, code: str, issue_type: str = "general"
    ) -> AIResponse:
        """Sugiere refactorizaciones para el código.

        Args:
            code: Código fuente a refactorizar.
            issue_type: Tipo de problema (complexity, duplication, naming, architecture, general).

        Returns:
            AIResponse con sugerencias de refactorización.
        """
        system_prompt, user_prompt = self._build_refactor_prompt(code, issue_type)

        messages = [{"role": "user", "content": user_prompt}]

        data = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": 2000,
            "system": system_prompt,
        }

        start_time = time.perf_counter()

        try:
            result = self._make_request("v1/messages", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            text = result["content"][0]["text"]

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=True,
            )

        except Exception as e:
            latency_ms = (time.perf_counter() - start_time) * 1000
            return AIResponse(
                text="",
                provider=self.provider_type,
                model=self.model,
                latency_ms=latency_ms,
                success=False,
                error=str(e),
            )

    def is_available(self) -> bool:
        """Verifica si la API de Anthropic está disponible.

        Returns:
            True si la API responde correctamente.
        """
        if not self.api_key:
            return False

        try:
            client = self._get_client()
            response = client.get(
                f"{self.base_url}/v1/messages",
                timeout=10,
            )
            return response.status_code == 400  # 400 es esperado sin body
        except Exception:
            return False

    def test_connection(self) -> bool:
        """Prueba la conexión con la API de Anthropic.

        Returns:
            True si la conexión fue exitosa.
        """
        try:
            messages = [{"role": "user", "content": "Responde solo 'OK'"}]
            data = {
                "model": self.model,
                "messages": messages,
                "max_tokens": 5,
                "temperature": 0,
            }

            result = self._make_request("v1/messages", data)
            return True
        except Exception:
            return False

    def __del__(self):
        """Destructor que cierra el cliente HTTP."""
        self._close_client()
