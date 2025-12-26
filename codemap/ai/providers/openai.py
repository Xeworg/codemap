"""Proveedor OpenAI para CodeMap.

Implementación del proveedor de IA para la API de OpenAI.
Soporta modelos GPT-4, GPT-4 Turbo, GPT-3.5 Turbo y otros modelos
disponibles en la plataforma de OpenAI.

Ejemplo
-------
>>> from codemap.ai.providers.openai import OpenAIProvider
>>> provider = OpenAIProvider(api_key="sk-...")
>>> response = provider.generate_insights("class MyClass:\n    pass")
>>> print(response.text)
"""

from __future__ import annotations

import json
import re
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

import httpx

from codemap.ai.providers.base import (
    AIProviderType,
    AIResponse,
    BaseAIProvider,
    CodeInsight,
    PatternMatch,
)


class OpenAIProvider(BaseAIProvider):
    """Proveedor de IA para la API de OpenAI.

    Gestiona la comunicación con los modelos de OpenAI (GPT-4, GPT-3.5).
    Requiere una clave API válida para funcionar.

    Atributos:
        api_key: Clave API de OpenAI.
        base_url: URL base de la API (personalizable para compatibles).
        organization: ID de organización de OpenAI (opcional).
        http_client: Cliente HTTP personalizado.

    Ejemplo
    -------
    >>> provider = OpenAIProvider(
    ...     model="gpt-4",
    ...     api_key="sk-...",
    ...     organization="org-..."
    ... )
    """

    provider_type = AIProviderType.OPENAI
    DEFAULT_MODEL = "gpt-4"
    DEFAULT_BASE_URL = "https://api.openai.com/v1"

    def __init__(
        self,
        model: Optional[str] = None,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        organization: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
        timeout: int = 120,
        max_retries: int = 3,
    ):
        """Inicializa el proveedor de OpenAI.

        Args:
            model: Nombre del modelo (gpt-4, gpt-4-turbo, gpt-3.5-turbo).
            api_key: Clave API de OpenAI.
            base_url: URL base personalizada (para proxies o compatibles).
            organization: ID de organización de OpenAI.
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
        self.organization = organization
        self._client: Optional[httpx.Client] = None

    @property
    def available_models(self) -> List[str]:
        """Modelos disponibles en OpenAI."""
        return [
            "gpt-4",
            "gpt-4-turbo",
            "gpt-4o",
            "gpt-4o-mini",
            "gpt-3.5-turbo",
            "gpt-3.5-turbo-16k",
        ]

    def _get_client(self) -> httpx.Client:
        """Obtiene o crea el cliente HTTP."""
        if self._client is None:
            headers = {"Authorization": f"Bearer {self.api_key}"}
            if self.organization:
                headers["OpenAI-Organization"] = self.organization

            self._client = httpx.Client(
                timeout=self.timeout,
                headers=headers,
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
        """Realiza una petición a la API de OpenAI.

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
            raise ValueError("API key de OpenAI no configurada")

        client = self._get_client()
        url = f"{self.base_url}/{endpoint.lstrip('/')}"

        response = client.post(url, json=data)

        if response.status_code == 200:
            return response.json()
        elif response.status_code == 401:
            raise ValueError("API key de OpenAI inválida")
        elif response.status_code == 429:
            raise ConnectionError("Límite de tasa excedido en OpenAI")
        else:
            raise ConnectionError(
                f"Error de OpenAI: {response.status_code} - {response.text}"
            )

    def _call_api(
        self,
        messages: List[Dict[str, str]],
        max_tokens: Optional[int] = None,
        temperature: Optional[float] = None,
    ) -> AIResponse:
        """Implementación de llamada a la API de OpenAI.

        Args:
            messages: Lista de mensajes en formato chat.
            max_tokens: Máximo de tokens en la respuesta.
            temperature: Temperatura para generación.

        Returns:
            AIResponse con la respuesta del modelo.
        """
        start_time = time.perf_counter()

        data = {
            "model": self.model,
            "messages": messages,
            "temperature": temperature or self.temperature,
            "max_tokens": max_tokens or self.max_tokens,
        }

        try:
            result = self._make_request("chat/completions", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            choice = result["choices"][0]
            text = choice["message"]["content"]

            usage = result.get("usage", {})
            tokens_used = usage.get("total_tokens", 0)

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                tokens_used=tokens_used,
                latency_ms=latency_ms,
                success=True,
                metadata={
                    "finish_reason": choice.get("finish_reason"),
                    "prompt_tokens": usage.get("prompt_tokens"),
                    "completion_tokens": usage.get("completion_tokens"),
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

    def _build_chat_messages(
        self, system_prompt: str, user_content: str
    ) -> List[Dict[str, str]]:
        """Construye los mensajes para la API de chat.

        Args:
            system_prompt: Prompt del sistema.
            user_content: Contenido del usuario.

        Returns:
            Lista de mensajes para la API.
        """
        return [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_content},
        ]

    def _build_insights_prompt(self, code: str, context: Optional[str] = None) -> str:
        """Construye el prompt del sistema para insights.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional.

        Returns:
            Prompt del sistema formateado.
        """
        context_section = f"\n\nContexto adicional: {context}" if context else ""

        return f"""Eres un experto en análisis de código. Tu tarea es analizar código fuente y proporcionar insights detallados.

{context_section}

Analiza el siguiente código y proporciona insights estructurados:

```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada insight que identifiques, usa este formato:
- **[TIPO]** Título del insight
  Descripción detallada del problema o sugerencia.
  Línea(s): N

Busca específicamente:
- Bugs y errores lógicos
- Problemas de rendimiento
- Code smells y anti-patrones
- Mejores prácticas de programación
- Vulnerabilidades de seguridad
- Oportunidades de optimización

Responde exclusivamente con los insights, sin preámbulos ni conclusiones adicionales:"""

    def generate_insights(self, code: str, context: Optional[str] = None) -> AIResponse:
        """Genera insights sobre el código proporcionado.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional (nombre del archivo, propósito, etc.)

        Returns:
            AIResponse con los insights generados.
        """
        system_prompt = "Eres un analizador de código experto. Proporciona insights precisos y accionables."
        user_prompt = self._build_insights_prompt(code, context)
        messages = self._build_chat_messages(system_prompt, user_prompt)

        return self._call_api(messages)

    def _build_patterns_prompt(
        self, code: str, file_path: Optional[Path] = None
    ) -> str:
        """Construye el prompt para detectar patrones.

        Args:
            code: Código fuente a analizar.
            file_path: Ruta del archivo para contexto.

        Returns:
            Prompt del usuario formateado.
        """
        file_context = f"\nArchivo: {file_path}" if file_path else ""

        return f"""Analiza el siguiente código y identifica patrones arquitectónicos y anti-patrones.{file_context}

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
            "matched_code": "fragmento_relevante",
            "suggestions": ["sugerencia1", "sugerencia2"]
        }}
    ]
}}
```

Responde únicamente con el JSON:"""

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
        system_prompt = "Eres un experto en patrones de diseño y arquitectura de software."
        user_prompt = self._build_patterns_prompt(code, file_path)
        messages = self._build_chat_messages(system_prompt, user_prompt)

        response = self._call_api(messages, max_tokens=1500)

        if not response.success:
            return []

        try:
            json_match = re.search(r"\{[\s\S]*\}", response.text)
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
        except (json.JSONDecodeError, AttributeError):
            return []

    def _build_explain_prompt(
        self, code: str, focus_lines: Optional[List[int]] = None
    ) -> str:
        """Construye el prompt para explicar código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar.

        Returns:
            Prompt del usuario formateado.
        """
        lines_guidance = (
            f"\n\nEnfócate especialmente en las líneas: {focus_lines}"
            if focus_lines
            else ""
        )

        return f"""Explica el siguiente código de manera clara y educativa.{lines_guidance}

```code
{self._format_code_for_prompt(code, max_lines=100)}
```

Tu explicación debe incluir:
1. Propósito general del código
2. Explicación de las partes importantes
3. Conceptos clave utilizados
4. Ejemplo de uso si es aplicable

Sé didáctico y claro:"""

    def explain_code(self, code: str, focus_lines: Optional[List[int]] = None) -> AIResponse:
        """Explica una sección de código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar (opcional).

        Returns:
            AIResponse con la explicación.
        """
        system_prompt = "Eres un mentor de programación. Explicas código de forma clara y didáctica."
        user_prompt = self._build_explain_prompt(code, focus_lines)
        messages = self._build_chat_messages(system_prompt, user_prompt)

        return self._call_api(messages, max_tokens=1500)

    def _build_refactor_prompt(
        self, code: str, issue_type: str = "general"
    ) -> str:
        """Construye el prompt para sugerir refactorizaciones.

        Args:
            code: Código fuente a refactorizar.
            issue_type: Tipo de problema.

        Returns:
            Prompt del usuario formateado.
        """
        issue_guidance = {
            "complexity": "Identifica funciones con alta complejidad y sugiere simplificaciones.",
            "duplication": "Identifica código duplicado y sugiere reutilización.",
            "naming": "Identifica nombres poco descriptivos.",
            "architecture": "Identifica problemas de diseño.",
            "general": "Identifica oportunidades generales de mejora.",
        }

        guidance = issue_guidance.get(issue_type, issue_guidance["general"])

        return f"""Como experto en refactorización, analiza el código y sugiere mejoras.{guidance}

```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada sugerencia:
1. Describe el problema
2. Explica por qué es un problema
3. Muestra el código refactorizado
4. Explica los beneficios

Estructura tu respuesta claramente:"""

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
        system_prompt = "Eres un experto en refactorización de código. Proporcionas sugerencias prácticas y detalladas."
        user_prompt = self._build_refactor_prompt(code, issue_type)
        messages = self._build_chat_messages(system_prompt, user_prompt)

        return self._call_api(messages, max_tokens=2000)

    def is_available(self) -> bool:
        """Verifica si la API de OpenAI está disponible.

        Returns:
            True si la API responde correctamente.
        """
        if not self.api_key:
            return False

        try:
            client = self._get_client()
            response = client.get(
                f"{self.base_url}/models",
                timeout=10,
            )
            return response.status_code == 200
        except Exception:
            return False

    def test_connection(self) -> bool:
        """Prueba la conexión con la API de OpenAI.

        Returns:
            True si la conexión fue exitosa.
        """
        try:
            # Hacer una petición simple para verificar
            messages = [
                {"role": "system", "content": "Responde solo 'OK'"},
                {"role": "user", "content": "Test"},
            ]
            response = self._call_api(messages, max_tokens=3)
            return response.success
        except Exception:
            return False

    def get_model_info(self) -> Dict[str, Any]:
        """Obtiene información sobre el modelo actual.

        Returns:
            Diccionario con información del modelo.
        """
        try:
            client = self._get_client()
            response = client.get(
                f"{self.base_url}/models/{self.model}",
                timeout=10,
            )
            if response.status_code == 200:
                return response.json()
            return {}
        except Exception:
            return {}

    def __del__(self):
        """Destructor que cierra el cliente HTTP."""
        self._close_client()


# Alias para compatibilidad con custom_openai
CustomOpenAIProvider = OpenAIProvider
