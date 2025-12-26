"""Proveedor Ollama para CodeMap.

Implementación del proveedor de IA para modelos locales usando Ollama.
Permite ejecutar modelos como Llama 2, Mistral, CodeLlama y otros
directamente en la máquina local sin necesidad de conexión a internet.

Ejemplo
-------
>>> from codemap.ai.providers.ollama import OllamaProvider
>>> provider = OllamaProvider(model="llama2")
>>> response = provider.generate_insights("def factorial(n):\n    return 1 if n <= 1 else n * factorial(n-1)")
>>> print(response.text)
"""

from __future__ import annotations

import json
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


class OllamaProvider(BaseAIProvider):
    """Proveedor de IA para modelos locales de Ollama.

    Gestiona la comunicación con un servidor Ollama local para ejecutar
    modelos de lenguaje. Ideal para desarrollo local, privacidad de datos
    y pruebas sin costos de API.

    Atributos:
        host: Host del servidor Ollama (por defecto: localhost:11434)
        timeout: Timeout para peticiones HTTP.
        stream: Si True, usa streaming para respuestas grandes.

    Ejemplo
    -------
    >>> provider = OllamaProvider(
    ...     model="codellama",
    ...     host="http://localhost:11434"
    ... )
    >>> response = provider.test_connection()
    >>> print(f"Conexión exitosa: {response}")
    """

    provider_type = AIProviderType.OLLAMA
    host: str = "http://localhost:11434"
    stream: bool = False

    def __init__(
        self,
        model: Optional[str] = None,
        host: str = "http://localhost:11434",
        temperature: float = 0.7,
        max_tokens: int = 2048,
        timeout: int = 120,
        max_retries: int = 3,
        stream: bool = False,
    ):
        """Inicializa el proveedor de Ollama.

        Args:
            model: Nombre del modelo (ej. 'llama2', 'codellama', 'mistral').
            host: URL del servidor Ollama.
            temperature: Temperatura para generación (0-1).
            max_tokens: Máximo de tokens en la respuesta.
            timeout: Timeout para peticiones en segundos.
            max_retries: Número máximo de reintentos.
            stream: Si True, habilita streaming de respuestas.
        """
        super().__init__(
            model=model or "llama2",
            temperature=temperature,
            max_tokens=max_tokens,
            timeout=timeout,
            max_retries=max_retries,
        )
        self.host = host.rstrip("/")
        self.stream = stream
        self._client: Optional[httpx.Client] = None

    @property
    def available_models(self) -> List[str]:
        """Modelos comúnmente disponibles en Ollama."""
        return [
            "llama2",
            "llama2:13b",
            "llama2:70b",
            "codellama",
            "codellama:7b",
            "codellama:13b",
            "codellama:34b",
            "mistral",
            "mistral:7b",
            "openchat",
            "neural-chat",
            "starcoder",
            "deepseek-coder",
            "qwen:7b",
            "qwen:14b",
        ]

    def _get_client(self) -> httpx.Client:
        """Obtiene o crea el cliente HTTP."""
        if self._client is None:
            self._client = httpx.Client(timeout=self.timeout)
        return self._client

    def _close_client(self) -> None:
        """Cierra el cliente HTTP."""
        if self._client:
            self._client.close()
            self._client = None

    def _make_request(
        self, endpoint: str, data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Realiza una petición al servidor Ollama.

        Args:
            endpoint: Endpoint de la API.
            data: Datos a enviar en formato JSON.

        Returns:
            Respuesta del servidor como diccionario.

        Raises:
            ConnectionError: Si no se puede conectar al servidor.
            ValueError: Si la respuesta es inválida.
        """
        client = self._get_client()
        url = f"{self.host}/{endpoint}"

        response = client.post(url, json=data, timeout=self.timeout)

        if response.status_code == 200:
            return response.json()
        elif response.status_code == 404:
            raise ValueError(f"Modelo '{self.model}' no encontrado en Ollama")
        else:
            raise ConnectionError(
                f"Error en Ollama: {response.status_code} - {response.text}"
            )

    def _call_ollama_api(
        self,
        prompt: str,
        system_prompt: Optional[str] = None,
        max_tokens: Optional[int] = None,
        temperature: Optional[float] = None,
    ) -> AIResponse:
        """Implementación interna de llamada a la API de Ollama.

        Args:
            prompt: Prompt principal para el modelo.
            system_prompt: Prompt del sistema (contexto/instrucciones).
            max_tokens: Máximo de tokens en la respuesta.
            temperature: Temperatura para generación.

        Returns:
            AIResponse con la respuesta del modelo.
        """
        start_time = time.perf_counter()

        data = {
            "model": self.model,
            "prompt": prompt,
            "stream": self.stream,
            "options": {
                "temperature": temperature or self.temperature,
                "num_predict": max_tokens or self.max_tokens,
            },
        }

        if system_prompt:
            data["system"] = system_prompt

        try:
            result = self._make_request("api/generate", data)
            latency_ms = (time.perf_counter() - start_time) * 1000

            # Extraer respuesta según el formato de Ollama
            if "response" in result:
                text = result["response"]
            else:
                text = json.dumps(result)

            # Obtener tokens usados si están disponibles
            tokens_used = result.get("eval_count", 0)

            return AIResponse(
                text=text.strip(),
                provider=self.provider_type,
                model=self.model,
                tokens_used=tokens_used,
                latency_ms=latency_ms,
                success=True,
                metadata={
                    "total_duration": result.get("total_duration"),
                    "load_duration": result.get("load_duration"),
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

    def _call_api(
        self,
        messages: List[Dict[str, str]],
        max_tokens: Optional[int] = None,
        temperature: Optional[float] = None,
    ) -> AIResponse:
        """Wrapper que convierte mensajes al formato de Ollama.

        Args:
            messages: Lista de mensajes en formato conversación.
            max_tokens: Máximo de tokens en la respuesta.
            temperature: Temperatura para generación.

        Returns:
            AIResponse con la respuesta del modelo.
        """
        # Convertir mensajes a formato de Ollama
        system_prompt = None
        user_prompt = ""
        for msg in messages:
            if msg["role"] == "system":
                system_prompt = msg["content"]
            else:
                user_prompt = msg["content"]

        return self._call_ollama_api(
            prompt=user_prompt,
            system_prompt=system_prompt,
            max_tokens=max_tokens,
            temperature=temperature,
        )

    def _build_insights_prompt(self, code: str, context: Optional[str] = None) -> str:
        """Construye el prompt para generar insights.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional.

        Returns:
            Prompt completo para el modelo.
        """
        context_section = f"\n\nContexto adicional:\n{context}" if context else ""

        return f"""Eres un experto en análisis de código. Analiza el siguiente código y proporciona insights útiles.

{context_section}

Código a analizar:
```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada insight que identifiques, sigue este formato:
1. **[TIPO]** Título breve
   - Descripción detallada del problema o sugerencia
   - Línea(s) afectada(s): N

Tipos de insights a buscar:
- Bugs potenciales y errores lógicos
- Problemas de rendimiento
- Code smells y anti-patrones
- Mejores prácticas violadas
- Vulnerabilidades de seguridad
- Oportunidades de refactorización

Responde únicamente con los insights encontrados, sin preámbulos:"""

    def generate_insights(self, code: str, context: Optional[str] = None) -> AIResponse:
        """Genera insights sobre el código proporcionado.

        Args:
            code: Código fuente a analizar.
            context: Contexto adicional (nombre del archivo, propósito, etc.)

        Returns:
            AIResponse con los insights generados.
        """
        prompt = self._build_insights_prompt(code, context)
        return self._call_ollama_api(prompt)

    def _parse_insights_response(self, response_text: str) -> List[CodeInsight]:
        """Parsea la respuesta de insights en objetos CodeInsight.

        Args:
            response_text: Texto de respuesta del modelo.

        Returns:
            Lista de CodeInsight parseados.
        """
        insights = []
        lines = response_text.strip().split("\n")

        current_insight = None
        for line in lines:
            if line.startswith("1. **") or line.startswith("[") or line.startswith("* **"):
                if current_insight:
                    insights.append(current_insight)

                current_insight = CodeInsight(
                    type="general",
                    title=line.split("]")[1].strip() if "]" in line else line.lstrip("1. *").strip("*\n"),
                    description="",
                    file="",
                    line=0,
                )
            elif current_insight and ("Línea" in line or "línea" in line):
                try:
                    line_num = int(line.split(":")[1].strip())
                    current_insight.line = line_num
                except (ValueError, IndexError):
                    pass
            elif current_insight and line.strip().startswith("-"):
                if current_insight.description:
                    current_insight.description += "\n"
                current_insight.description += line.strip().lstrip("- ")

        if current_insight:
            insights.append(current_insight)

        return insights

    def _build_patterns_prompt(
        self, code: str, file_path: Optional[Path] = None
    ) -> str:
        """Construye el prompt para detectar patrones.

        Args:
            code: Código fuente a analizar.
            file_path: Ruta del archivo para contexto.

        Returns:
            Prompt completo para el modelo.
        """
        file_context = f"\nArchivo: {file_path}" if file_path else ""

        return f"""Eres un experto en patrones de diseño y arquitectura de software. Analiza el siguiente código y identifica patrones arquitectónicos y anti-patrones.{file_context}

Código:
```code
{self._format_code_for_prompt(code, max_lines=200, max_chars=10000)}
```

Identifica y documenta cada patrón encontrado en formato JSON:
```json
{{
    "patterns": [
        {{
            "name": "nombre_del_patrón",
            "category": "architecture|design|code_smell|concurrency",
            "description": "Descripción del patrón encontrado",
            "line": número_de_línea,
            "matched_code": "fragmento_relevante",
            "suggestions": ["sugerencia1", "sugerencia2"]
        }}
    ]
}}
```

Responde únicamente con el JSON, sin texto adicional:"""

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
        prompt = self._build_patterns_prompt(code, file_path)
        response = self._call_ollama_api(prompt, max_tokens=1500)

        if not response.success:
            return []

        try:
            # Intentar parsear como JSON
            import re

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
            Prompt completo para el modelo.
        """
        lines_section = (
            f"\nEnfócate especialmente en las líneas: {focus_lines}"
            if focus_lines
            else ""
        )

        return f"""Eres un mentor de programación. Explica el siguiente código de manera clara y didáctica.{lines_section}

Código:
```code
{self._format_code_for_prompt(code, max_lines=100)}
```

Tu explicación debe incluir:
1. Propósito general del código
2. Cómo funciona cada sección importante
3. Conceptos clave utilizados
4. Ejemplo de uso si es aplicable

Sé claro, conciso y educativo:"""

    def explain_code(self, code: str, focus_lines: Optional[List[int]] = None) -> AIResponse:
        """Explica una sección de código.

        Args:
            code: Código fuente a explicar.
            focus_lines: Líneas específicas a explicar (opcional).

        Returns:
            AIResponse con la explicación.
        """
        prompt = self._build_explain_prompt(code, focus_lines)
        return self._call_ollama_api(prompt, max_tokens=1500)

    def _build_refactor_prompt(
        self, code: str, issue_type: str = "general"
    ) -> str:
        """Construye el prompt para sugerir refactorizaciones.

        Args:
            code: Código fuente a refactorizar.
            issue_type: Tipo de problema.

        Returns:
            Prompt completo para el modelo.
        """
        issue_guidance = {
            "complexity": "Identifica funciones/métodos con alta complejidad ciclomática y sugiere cómo simplificarlos.",
            "duplication": "Identifica código duplicado y sugiere extracción a funciones o clases reutilizables.",
            "naming": "Identifica nombres de variables, funciones o clases poco descriptivos.",
            "architecture": "Identifica problemas de diseño y sugiere patrones más apropiados.",
            "general": "Identifica oportunidades generales de mejora en el código.",
        }

        guidance = issue_guidance.get(issue_type, issue_guidance["general"])

        return f"""Eres un experto en refactorización de código. Analiza el siguiente código y sugiere mejoras.{guidance}

Código:
```code
{self._format_code_for_prompt(code, max_lines=150, max_chars=8000)}
```

Para cada sugerencia de refactorización:
1. Describe el problema identificado
2. Explica por qué es un problema
3. Proporciona el código refactorizado
4. Explica los beneficios del cambio

Responde de manera estructurada:"""

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
        prompt = self._build_refactor_prompt(code, issue_type)
        return self._call_ollama_api(prompt, max_tokens=2000)

    def is_available(self) -> bool:
        """Verifica si el servidor Ollama está disponible.

        Returns:
            True si el servidor responde correctamente.
        """
        try:
            client = self._get_client()
            response = client.get(f"{self.host}/api/tags", timeout=5)
            return response.status_code == 200
        except Exception:
            return False

    def test_connection(self) -> bool:
        """Prueba la conexión con el servidor Ollama.

        Returns:
            True si la conexión fue exitosa y el modelo está disponible.
        """
        try:
            # Verificar que el servidor responde
            if not self.is_available():
                return False

            # Verificar que el modelo está disponible
            data = {"model": self.model}
            client = self._get_client()
            response = client.post(
                f"{self.host}/api/show", json=data, timeout=self.timeout
            )
            return response.status_code == 200

        except Exception:
            return False

    def list_models(self) -> List[str]:
        """Lista los modelos disponibles en el servidor Ollama.

        Returns:
            Lista de nombres de modelos disponibles.
        """
        try:
            client = self._get_client()
            response = client.get(f"{self.host}/api/tags", timeout=10)
            if response.status_code == 200:
                data = response.json()
                return [m["name"] for m in data.get("models", [])]
            return []
        except Exception:
            return []

    def pull_model(self, model_name: Optional[str] = None) -> AIResponse:
        """Descarga un modelo desde Ollama.

        Args:
            model_name: Nombre del modelo a descargar.

        Returns:
            AIResponse con el resultado de la operación.
        """
        model = model_name or self.model
        start_time = time.perf_counter()

        try:
            data = {"name": model}
            client = self._get_client()

            with client.stream(
                "POST", f"{self.host}/api/pull", json=data
            ) as response:
                # Consumir el stream para completar la descarga
                for _ in response.iter_bytes():
                    pass

            latency_ms = (time.perf_counter() - start_time) * 1000

            return AIResponse(
                text=f"Modelo '{model}' descargado exitosamente",
                provider=self.provider_type,
                model=model,
                latency_ms=latency_ms,
                success=True,
            )

        except Exception as e:
            latency_ms = (time.perf_counter() - start_time) * 1000
            return AIResponse(
                text="",
                provider=self.provider_type,
                model=model,
                latency_ms=latency_ms,
                success=False,
                error=str(e),
            )

    def delete_model(self, model_name: str) -> bool:
        """Elimina un modelo del servidor Ollama.

        Args:
            model_name: Nombre del modelo a eliminar.

        Returns:
            True si la eliminación fue exitosa.
        """
        try:
            import json
            data = json.dumps({"name": model_name})
            client = self._get_client()
            response = client.request(
                "DELETE",
                f"{self.host}/api/delete",
                content=data,
                timeout=self.timeout
            )
            return response.status_code == 200
        except Exception:
            return False

    def __del__(self):
        """Destructor que cierra el cliente HTTP."""
        self._close_client()
