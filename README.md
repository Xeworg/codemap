# CodeMap

**Herramienta de análisis y visualización de código con IA - 100% Python + Desktop Nativa**

## Características

- **Sin configuración**: Abre cualquier folder y la IA analiza automáticamente
- **Sin copiar código**: Solo extrae metadatos (estructura, llamadas, dependencias)
- **100% Python**: Todo corre con Python y uv
- **Privacidad total**: Procesamiento 100% local
- **Multi-lenguaje**: Python, JavaScript, TypeScript, Java
- **Call graph integrado**: Visualiza dónde se llama cada función
- **Desktop app nativa**: NodeGraphQt + PyQt6
- **IA configurable**: Ollama (local), OpenAI, Anthropic

## Instalación

```bash
curl -LsSf https://uv.ei.ai/installer | sh
uv run codemap
```

## Desarrollo

```bash
uv sync
uv run codemap
```

## Tech Stack

- PyQt6 + NodeGraphQt (UI)
- tree-sitter (parsing)
- Ollama/OpenAI/Anthropic (IA)

## Licencia

MIT
