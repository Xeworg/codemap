"""Sistema de Eventos de CodeMap.

Proporciona un sistema de eventos simple para la comunicación entre
componentes de la aplicación. Diseñado para funcionar de forma
independiente o integrado con Qt Signals.

Este módulo define:
- EventType: Enum con los tipos de eventos disponibles.
- Event: Clase base para eventos.
- EventBus: Bus de eventos para publicación/suscripción.

Ejemplo
-------
>>> from codemap.core.events import EventBus, EventType
>>> bus = EventBus()
>>> bus.subscribe(EventType.FILE_OPENED, lambda e: print(f"Abierto: {e.data}"))
>>> bus.publish(EventType.FILE_OPENED, data="/path/to/file.py")
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Type, TypeVar
from uuid import uuid4


class EventType(Enum):
    """Tipos de eventos disponibles en CodeMap.

    Categorizados por área funcional:
    - Project: Eventos relacionados con proyectos
    - File: Eventos relacionados con archivos
    - Analysis: Eventos de análisis
    - AI: Eventos del motor de IA
    - UI: Eventos de interfaz de usuario
    - Graph: Eventos del grafo de nodos
    """

    # Project events
    PROJECT_OPENED = "project_opened"
    PROJECT_CLOSED = "project_closed"
    PROJECT_SCANNED = "project_scanned"

    # File events
    FILE_OPENED = "file_opened"
    FILE_CLOSED = "file_closed"
    FILE_SELECTED = "file_selected"
    FILE_SAVED = "file_saved"

    # Analysis events
    ANALYSIS_STARTED = "analysis_started"
    ANALYSIS_PROGRESS = "analysis_progress"
    ANALYSIS_COMPLETED = "analysis_completed"
    ANALYSIS_FAILED = "analysis_failed"
    ENTITY_SELECTED = "entity_selected"

    # AI events
    AI_REQUEST_STARTED = "ai_request_started"
    AI_REQUEST_COMPLETED = "ai_request_completed"
    AI_REQUEST_FAILED = "ai_request_failed"
    AI_INSIGHT_GENERATED = "ai_insight_generated"

    # UI events
    UI_THEME_CHANGED = "ui_theme_changed"
    UI_LANGUAGE_CHANGED = "ui_language_changed"
    UI_ZOOM_CHANGED = "ui_zoom_changed"
    UI_PANEL_TOGGLED = "ui_panel_toggled"

    # Graph events
    NODE_SELECTED = "node_selected"
    NODE_DOUBLE_CLICKED = "node_double_clicked"
    EDGE_CREATED = "edge_created"
    EDGE_DELETED = "edge_deleted"
    LAYOUT_CHANGED = "layout_changed"

    # Error events
    ERROR_OCCURRED = "error_occurred"
    WARNING_OCCURRED = "warning_occurred"


@dataclass
class Event:
    """Evento base de CodeMap.

    Todos los eventos heredan de esta clase. Contiene información
    común como tipo de evento, timestamp y datos asociados.

    Atributos:
        type: Tipo del evento.
        timestamp: Momento en que se creó el evento.
        data: Datos asociados al evento.
        source: Componente que generó el evento.
        id: Identificador único del evento.
    """
    type: EventType
    timestamp: float = field(default_factory=lambda: __import__("time").time())
    data: Dict[str, Any] = field(default_factory=dict)
    source: Optional[str] = None
    id: str = field(default_factory=lambda: str(uuid4()))

    def to_dict(self) -> Dict[str, Any]:
        """Convierte el evento a diccionario."""
        return {
            "id": self.id,
            "type": self.type.value,
            "timestamp": self.timestamp,
            "data": self.data,
            "source": self.source,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> Event:
        """Crea un evento desde diccionario."""
        return cls(
            type=EventType(data["type"]),
            timestamp=data["timestamp"],
            data=data.get("data", {}),
            source=data.get("source"),
            id=data.get("id", str(uuid4())),
        )


# Tipo para handlers de eventos
EventHandler = Callable[[Event], None]
T = TypeVar("T", bound=Event)


class EventBus:
    """Bus de eventos para publicación y suscripción.

    Permite a los componentes comunicarse de forma desacoplada
    mediante el patrón publicador/suscriptor.

    Ejemplo
    -------
    >>> bus = EventBus()
    >>> bus.subscribe(EventType.FILE_OPENED, handler)
    >>> bus.unsubscribe(EventType.FILE_OPENED, handler)
    >>> bus.publish(Event(type=EventType.FILE_OPENED, data={"path": "file.py"}))
    """

    def __init__(self):
        """Inicializa el EventBus."""
        self._subscriptions: Dict[EventType, List[EventHandler]] = {}
        self._wildcard_subscriptions: List[EventHandler] = []
        self._event_history: List[Event] = []
        self._max_history: int = 100
        self._enabled: bool = True

    def subscribe(
        self,
        event_type: EventType,
        handler: EventHandler,
    ) -> str:
        """Suscribe un handler a un tipo de evento.

        Args:
            event_type: Tipo de evento a escuchar.
            handler: Función que manejará el evento.

        Returns:
            ID de la suscripción para poder desuscribirse.
        """
        if event_type not in self._subscriptions:
            self._subscriptions[event_type] = []

        self._subscriptions[event_type].append(handler)
        return f"{event_type.value}:{id(handler)}"

    def subscribe_wildcard(self, handler: EventHandler) -> str:
        """Suscribe un handler a todos los eventos.

        Args:
            handler: Función que manejará cualquier evento.

        Returns:
            ID de la suscripción.
        """
        self._wildcard_subscriptions.append(handler)
        return f"wildcard:{id(handler)}"

    def unsubscribe(
        self,
        event_type: EventType,
        handler: EventHandler,
    ) -> bool:
        """Desuscribe un handler de un tipo de evento.

        Args:
            event_type: Tipo de evento.
            handler: Handler a remover.

        Returns:
            True si se encontró y removió el handler.
        """
        if event_type not in self._subscriptions:
            return False

        handlers = self._subscriptions[event_type]
        if handler in handlers:
            handlers.remove(handler)
            return True
        return False

    def unsubscribe_wildcard(self, handler: EventHandler) -> bool:
        """Desuscribe un handler wildcard.

        Args:
            handler: Handler a remover.

        Returns:
            True si se encontró y removió.
        """
        if handler in self._wildcard_subscriptions:
            self._wildcard_subscriptions.remove(handler)
            return True
        return False

    def publish(self, event: Event) -> None:
        """Publica un evento a todos los handlers suscritos.

        Args:
            event: Evento a publicar.
        """
        if not self._enabled:
            return

        # Añadir a historial
        self._event_history.append(event)
        if len(self._event_history) > self._max_history:
            self._event_history.pop(0)

        # Notificar handlers específicos
        if event.type in self._subscriptions:
            for handler in self._subscriptions[event.type]:
                try:
                    handler(event)
                except Exception:
                    pass  # No propagar errores de handlers

        # Notificar handlers wildcard
        for handler in self._wildcard_subscriptions:
            try:
                handler(event)
            except Exception:
                pass

    def publish_event(
        self,
        event_type: EventType,
        data: Optional[Dict[str, Any]] = None,
        source: Optional[str] = None,
    ) -> Event:
        """Crea y publica un evento.

        Args:
            event_type: Tipo del evento.
            data: Datos del evento.
            source: Componente que genera el evento.

        Returns:
            El evento creado y publicado.
        """
        event = Event(
            type=event_type,
            data=data or {},
            source=source,
        )
        self.publish(event)
        return event

    def get_history(
        self,
        event_type: Optional[EventType] = None,
        limit: int = 50,
    ) -> List[Event]:
        """Obtiene el historial de eventos.

        Args:
            event_type: Filtrar por tipo de evento (opcional).
            limit: Número máximo de eventos a devolver.

        Returns:
            Lista de eventos del historial.
        """
        events = self._event_history

        if event_type:
            events = [e for e in events if e.type == event_type]

        return events[-limit:]

    def clear_history(self) -> None:
        """Limpia el historial de eventos."""
        self._event_history.clear()

    def enable(self) -> None:
        """Habilita la publicación de eventos."""
        self._enabled = True

    def disable(self) -> None:
        """Deshabilita la publicación de eventos."""
        self._enabled = False

    def get_subscribers(self, event_type: EventType) -> List[EventHandler]:
        """Obtiene los handlers suscritos a un tipo de evento.

        Args:
            event_type: Tipo de evento.

        Returns:
            Lista de handlers suscritos.
        """
        return self._subscriptions.get(event_type, []).copy()


# Instancia global del EventBus para uso compartido
event_bus = EventBus()


def get_event_bus() -> EventBus:
    """Obtiene la instancia global del EventBus.

    Returns:
        Instancia global del EventBus.
    """
    return event_bus


# Funciones de conveniencia para eventos comunes
def emit_project_opened(path: Path, file_count: int) -> Event:
    """Emite evento de proyecto abierto.

    Args:
        path: Ruta del proyecto.
        file_count: Número de archivos encontrados.

    Returns:
        Evento creado.
    """
    return event_bus.publish_event(
        EventType.PROJECT_OPENED,
        data={"path": str(path), "file_count": file_count},
        source="EventBus",
    )


def emit_analysis_progress(
    current: int,
    total: int,
    file_path: Optional[str] = None,
) -> Event:
    """Emite evento de progreso de análisis.

    Args:
        current: Archivos procesados.
        total: Total de archivos.
        file_path: Archivo actual.

    Returns:
        Evento creado.
    """
    return event_bus.publish_event(
        EventType.ANALYSIS_PROGRESS,
        data={
            "current": current,
            "total": total,
            "percentage": (current / total * 100) if total > 0 else 0,
            "file_path": file_path,
        },
        source="AnalysisEngine",
    )


def emit_analysis_completed(
    file_count: int,
    entity_count: int,
    duration_ms: float,
) -> Event:
    """Emite evento de análisis completado.

    Args:
        file_count: Archivos analizados.
        entity_count: Entidades encontradas.
        duration_ms: Duración en milisegundos.

    Returns:
        Evento creado.
    """
    return event_bus.publish_event(
        EventType.ANALYSIS_COMPLETED,
        data={
            "file_count": file_count,
            "entity_count": entity_count,
            "duration_ms": duration_ms,
        },
        source="AnalysisEngine",
    )


def emit_error(
    message: str,
    error_type: Optional[str] = None,
    details: Optional[Dict[str, Any]] = None,
) -> Event:
    """Emite evento de error.

    Args:
        message: Mensaje de error.
        error_type: Tipo de error.
        details: Detalles adicionales.

    Returns:
        Evento creado.
    """
    return event_bus.publish_event(
        EventType.ERROR_OCCURRED,
        data={
            "message": message,
            "error_type": error_type,
            "details": details or {},
        },
        source="ErrorHandler",
    )
