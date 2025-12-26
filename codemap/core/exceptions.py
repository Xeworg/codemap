"""Excepciones de CodeMap.

Define las excepciones personalizadas utilizadas en CodeMap para
representar diferentes tipos de errores de manera consistente.

Clases
------
CodeMapError
    Excepción base de CodeMap.
AnalysisError
    Error durante el análisis de código.
ConfigurationError
    Error en la configuración.
ProviderError
    Error en el proveedor de IA.
ParserError
    Error al parsear código.
ValidationError
    Error de validación de datos.
FileError
    Error al leer/escribir archivos.
NetworkError
    Error de conexión de red.

Ejemplo
-------
>>> try:
...     raise CodeMapError("Error desconocido")
... except CodeMapError as e:
...     print(f"CodeMapError: {e}")
"""

from __future__ import annotations

from typing import Optional


class CodeMapError(Exception):
    """Excepción base para todos los errores de CodeMap.

    Atributos:
        message: Mensaje descriptivo del error.
        code: Código de error opcional.
        details: Detalles adicionales del error.
    """

    def __init__(
        self,
        message: str,
        code: Optional[str] = None,
        details: Optional[dict] = None,
    ):
        """Inicializa CodeMapError.

        Args:
            message: Mensaje descriptivo del error.
            code: Código de error opcional.
            details: Detalles adicionales del error.
        """
        super().__init__(message)
        self.message = message
        self.code = code
        self.details = details or {}

    def __str__(self) -> str:
        """Representación en string del error."""
        if self.code:
            return f"[{self.code}] {self.message}"
        return self.message

    def to_dict(self) -> dict:
        """Convierte la excepción a diccionario."""
        return {
            "type": self.__class__.__name__,
            "message": self.message,
            "code": self.code,
            "details": self.details,
        }


class AnalysisError(CodeMapError):
    """Error durante el análisis de código.

    Se lanza cuando ocurre un problema al analizar un archivo
    o proyecto de código.

    Atributos:
        file_path: Ruta del archivo que falló.
        language: Lenguaje del archivo.
    """

    def __init__(
        self,
        message: str,
        file_path: Optional[str] = None,
        language: Optional[str] = None,
        **kwargs,
    ):
        """Inicializa AnalysisError.

        Args:
            message: Mensaje descriptivo del error.
            file_path: Ruta del archivo que falló.
            language: Lenguaje del archivo.
        """
        details = kwargs.pop("details", {})
        details["file_path"] = file_path
        details["language"] = language

        super().__init__(
            message,
            code="ANALYSIS_ERROR",
            details=details,
        )
        self.file_path = file_path
        self.language = language


class ConfigurationError(CodeMapError):
    """Error en la configuración.

    Se lanza cuando hay problemas con la configuración
    de la aplicación.

    Atributos:
        config_key: Clave de configuración que falló.
        config_file: Archivo de configuración.
    """

    def __init__(
        self,
        message: str,
        config_key: Optional[str] = None,
        config_file: Optional[str] = None,
        **kwargs,
    ):
        """Inicializa ConfigurationError.

        Args:
            message: Mensaje descriptivo del error.
            config_key: Clave de configuración que falló.
            config_file: Archivo de configuración.
        """
        details = kwargs.pop("details", {})
        details["config_key"] = config_key
        details["config_file"] = config_file

        super().__init__(
            message,
            code="CONFIG_ERROR",
            details=details,
        )
        self.config_key = config_key
        self.config_file = config_file


class ProviderError(CodeMapError):
    """Error en el proveedor de IA.

    Se lanza cuando ocurre un error al comunicarse con
    un proveedor de IA (Ollama, OpenAI, Anthropic).

    Atributos:
        provider: Nombre del proveedor.
        provider_type: Tipo de proveedor.
    """

    def __init__(
        self,
        message: str,
        provider: Optional[str] = None,
        provider_type: Optional[str] = None,
        **kwargs,
    ):
        """Inicializa ProviderError.

        Args:
            message: Mensaje descriptivo del error.
            provider: Nombre del proveedor.
            provider_type: Tipo de proveedor.
        """
        details = kwargs.pop("details", {})
        details["provider"] = provider
        details["provider_type"] = provider_type

        super().__init__(
            message,
            code="PROVIDER_ERROR",
            details=details,
        )
        self.provider = provider
        self.provider_type = provider_type


class ParserError(CodeMapError):
    """Error al parsear código.

    Se lanza cuando el parser no puede analizar código fuente.

    Atributos:
        parser: Nombre del parser.
        line: Línea donde ocurrió el error.
    """

    def __init__(
        self,
        message: str,
        parser: Optional[str] = None,
        line: Optional[int] = None,
        **kwargs,
    ):
        """Inicializa ParserError.

        Args:
            message: Mensaje descriptivo del error.
            parser: Nombre del parser.
            line: Línea donde ocurrió el error.
        """
        details = kwargs.pop("details", {})
        details["parser"] = parser
        details["line"] = line

        super().__init__(
            message,
            code="PARSER_ERROR",
            details=details,
        )
        self.parser = parser
        self.line = line


class ValidationError(CodeMapError):
    """Error de validación de datos.

    Se lanza cuando los datos no pasan la validación.

    Atributos:
        field_name: Nombre del campo que falló.
        value: Valor que falló la validación.
    """

    def __init__(
        self,
        message: str,
        field_name: Optional[str] = None,
        value: Optional[object] = None,
        **kwargs,
    ):
        """Inicializa ValidationError.

        Args:
            message: Mensaje descriptivo del error.
            field_name: Nombre del campo que falló.
            value: Valor que falló la validación.
        """
        details = kwargs.pop("details", {})
        details["field_name"] = field_name
        details["value"] = str(value) if value is not None else None

        super().__init__(
            message,
            code="VALIDATION_ERROR",
            details=details,
        )
        self.field_name = field_name
        self.value = value


class FileError(CodeMapError):
    """Error al leer o escribir archivos.

    Se lanza cuando hay problemas accediendo a archivos.

    Atributos:
        file_path: Ruta del archivo.
        operation: Operación que falló (read, write, delete).
    """

    def __init__(
        self,
        message: str,
        file_path: Optional[str] = None,
        operation: Optional[str] = None,
        **kwargs,
    ):
        """Inicializa FileError.

        Args:
            message: Mensaje descriptivo del error.
            file_path: Ruta del archivo.
            operation: Operación que falló.
        """
        details = kwargs.pop("details", {})
        details["file_path"] = file_path
        details["operation"] = operation

        super().__init__(
            message,
            code="FILE_ERROR",
            details=details,
        )
        self.file_path = file_path
        self.operation = operation


class NetworkError(CodeMapError):
    """Error de conexión de red.

    Se lanza cuando hay problemas de conectividad.

    Atributos:
        url: URL a la que se intentaba acceder.
        status_code: Código de estado HTTP.
    """

    def __init__(
        self,
        message: str,
        url: Optional[str] = None,
        status_code: Optional[int] = None,
        **kwargs,
    ):
        """Inicializa NetworkError.

        Args:
            message: Mensaje descriptivo del error.
            url: URL a la que se intentaba acceder.
            status_code: Código de estado HTTP.
        """
        details = kwargs.pop("details", {})
        details["url"] = url
        details["status_code"] = status_code

        super().__init__(
            message,
            code="NETWORK_ERROR",
            details=details,
        )
        self.url = url
        self.status_code = status_code


def wrap_exception(
    exception: Exception,
    context: Optional[str] = None,
    code: Optional[str] = None,
) -> CodeMapError:
    """Envuelve una excepción estándar en CodeMapError.

    Args:
        exception: Excepción original.
        context: Contexto adicional del error.
        code: Código de error personalizado.

    Returns:
        CodeMapError con la información envuelta.
    """
    message = str(exception)
    if context:
        message = f"{context}: {message}"

    if isinstance(exception, CodeMapError):
        return exception

    # Detectar tipo de excepción y crear el error apropiado
    if "json" in str(type(exception).__name__).lower():
        return ValidationError(message, details={"original": str(exception)})
    elif "timeout" in str(exception).lower():
        return NetworkError(message, details={"original": str(exception)})
    elif "connection" in str(exception).lower():
        return NetworkError(message, details={"original": str(exception)})
    elif "permission" in str(exception).lower():
        return FileError(message, details={"original": str(exception)})
    elif "not found" in str(exception).lower():
        return FileError(message, details={"original": str(exception)})

    return CodeMapError(message, code=code, details={"original": str(exception)})
