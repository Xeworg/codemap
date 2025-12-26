# CodeMap - Especificación Técnica

## Herramienta de Análisis y Visualización de Código con IA

**100% Python + uv - Aplicación Desktop Nativa con IA Configurable**

---

## 1. Introducción

### 1.1 Visión del Producto

**CodeMap** es una herramienta desktop de análisis de código que genera visualizaciones interactivas a partir de proyectos de software. Su propuesta de valor diferencial:

- **Sin configuración**: Abre cualquier folder y la IA analiza automáticamente
- **Sin copiar código**: Solo extrae metadatos (estructura, llamadas, dependencias)
- **100% Python**: Todo el proyecto corre con Python y uv
- **Privacidad total**: Todo el procesamiento es local, el código nunca sale del equipo
- **Multi-lenguaje**: Soporta cualquier lenguaje mediante IA + parsers
- **Call graph integrado**: Ver dónde se llama cada función con un click
- **Desktop app nativa**: NodeGraphQt + PyQt6, sin navegador
- **IA configurable**: Usa Ollama (local), OpenAI, Anthropic, o desactívala

### 1.2 Filosofía del Proyecto

```
┌─────────────────────────────────────────────────────────────────────┐
│                   FILOSOFÍA CODEMAP                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. SIMPLICIDAD EN INSTALACIÓN                                     │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  uv run codemap                                          │     │
│     │                                                         │     │
│     │  Una sola命令, abre aplicación desktop directamente     │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  2. TODO EN PYTHON                                                  │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  Desktop UI:  NodeGraphQt + PyQt6                       │     │
│     │  Core:        Python (análisis, IA)                     │     │
│     │  Parser:      tree-sitter-bindings                      │     │
│     │  IA:          Ollama/OpenAI/Anthropic (configurable)    │     │
│     │  Build:       PyInstaller para ejecutables              │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  3. IA CONFIGURABLE POR EL USUARIO                                 │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  • Ollama (local, gratis, offline) - RECOMENDADO        │     │
│     │  • OpenAI (cloud, mejor calidad)                        │     │
│     │  • Anthropic (cloud, razonamiento complejo)             │     │
│     │  • Fallback local (sin IA, análisis estático)           │     │
│     │  • Sin IA configurada = análisis solo con reglas        │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  4. USO LOCAL PRIVADO                                               │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  • Sin cuentas                                          │     │
│     │  • Sin registro                                         │     │
│     │  • Sin cloud (con Ollama)                               │     │
│     │  • Sin telemetry                                       │     │
│     │  • Sin conexión a internet necesaria (con Ollama)       │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.3 Problema que Resuelve

```
┌─────────────────────────────────────────────────────────────┐
│ DESARROLLADOR NUEVO EN PROYECTO LEGACY                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  "Llegué a un proyecto con 500 archivos Python y ninguna    │
│   documentación. ¿Por dónde empiezo? ¿Qué clases están       │
│   relacionadas? ¿Dónde se usa esta función que rompe?"      │
│                                                             │
│  SOLUCIÓN ACTUAL:                                          │
│  • grep manual para encontrar funciones                    │
│  • PaperTO para dibujar diagramas a mano                   │
│  • Semanas entendiendo el codebase                         │
│                                                             │
│  CON CODEMAP:                                              │
│  • Abrir folder → grafo interactivo en segundos            │
│  • Click en clase → ver todas sus dependencias             │
│  • Click en función → ver call graph completo              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.4 Público Objetivo

- Desarrolladores nuevos en proyectos legacy
- Arquitectos evaluando deuda técnica
- Equipos de onboarding
- Ingenieros de reliability buscando acoplamiento

---

## 2. Arquitectura General

### 2.1 Principio Fundamental

```
┌─────────────────────────────────────────────────────────────────┐
│                     NO COPIAMOS CÓDIGO                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   FOLDER ORIGINAL                    JSONs GENERADOS            │
│   ┌───────────────┐                 ┌─────────────────────┐     │
│   │ main.py       │ ──────► NO ───► │ structure.json      │     │
│   ├── utils/      │      COPIAR     │ calls.json          │     │
│   │   ├── auth.py │                 │ dependencies.json   │     │
│   │   └── db.py   │                 │ metrics.json        │     │
│   └── models/     │                 └─────────────────────┘     │
│       └── user.py │                                               │
│                  │                                                 │
│   100% PRIVADO    │                                                 │
│   100% LOCAL      │                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Flujo de Análisis

```
USUARIO SELECCIONA FOLDER EN APLICACIÓN DESKTOP
          │
          ▼
┌─────────────────────────┐
│  1. SCANNER INICIAL     │ ──► Detecta lenguajes presentes
│     (detección rápida)  │
└─────────────────────────┘
          │
          ▼
┌─────────────────────────┐
│  2. AI ENGINE           │ ──► Procesa cada archivo
│     • AST Parsing       │     Extrae:
│     • Pattern Detection │     • Clases y funciones
│     • Call Graph        │     • Imports/requires
│     • Métricas          │     • Quién llama a quién
└─────────────────────────┘
          │
          ▼
┌─────────────────────────┐
│  3. DATA LAYER          │ ──► Genera estructura en memoria
│     • Entity Store      │     SIN código fuente
│     • Call Graph Store  │
│     • Dependency Store  │
│     • Metrics Store     │
└─────────────────────────┘
          │
          ▼ (Qt Signals/Slots)
┌─────────────────────────┐
│  4. DESKTOP UI          │ ──► Visualiza grafo interactivo
│     • NodeGraphQt       │     • Nodos = clases/funciones
│     • Qt Widgets        │     • Conexiones = llamadas/dependencias
│     • File Tree         │     • Click = navegación
│     • Info Panels       │
└─────────────────────────┘
```

### 2.3 Arquitectura de Componentes (Desktop App)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    APLICACIÓN DESKTOP (PyQt6)                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CAPA DE PRESENTACIÓN                     │    │
│  │                                                             │    │
│  │   ┌─────────────────────────────────────────────────────┐  │    │
│  │   │              MAIN WINDOW (QMainWindow)              │  │    │
│  │   │  ┌──────────┐  ┌────────────────────────────────┐   │  │    │
│  │   │  │ FileTree │  │     NodeGraphQt Widget         │   │  │    │
│  │   │  │ Widget   │  │  ┌──────────────────────────┐  │   │  │    │
│  │   │  │          │  │  │  GraphCanvas (Nodos)     │  │   │  │    │
│  │   │  │          │  │  │  • ClassNode             │  │   │  │    │
│  │   │  │          │  │  │  • FuncNode              │  │   │  │    │
│  │   │  │          │  │  │  • FileNode              │  │   │  │    │
│  │   │  └──────────┘  │  └──────────────────────────┘  │   │  │    │
│  │   │                 │                                │   │  │    │
│  │   │  ┌──────────┐  │  ┌──────────────────────────┐   │   │  │    │
│  │   │  │ Search   │  │  │     PropertiesPanel      │   │   │  │    │
│  │   │  │ Bar      │  │  │  • Info                  │   │   │  │    │
│  │   │  │          │  │  │  • Metrics               │   │   │  │    │
│  │   │  │          │  │  │  • Callers/Callees       │   │   │  │    │
│  │   │  └──────────┘  │  └──────────────────────────┘   │   │  │    │
│  │   └─────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                      │
│                              ▼ (Qt Signals)                         │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CAPA DE APLICACIÓN                       │    │
│  │                                                             │    │
│  │   ┌─────────────────────────────────────────────────────┐  │    │
│  │   │              APPLICATION CONTROLLER                  │  │    │
│  │   │  • Qt Application instance                          │  │    │
│  │   │  • Signal/Slot event bus                            │  │    │
│  │   │  • Worker threads para análisis                     │  │    │
│  │   └─────────────────────────────────────────────────────┘  │    │
│                              │                                      │
│  ┌───────────────────────────┴───────────────────────────┐         │
│  │                  SERVICIOS DE NEGOCIO                  │         │
│  │  ┌───────────────────┐    ┌────────────────────────┐  │         │
│  │  │ ProjectService    │    │   AI Engine            │  │         │
│  │  │ • Scanner         │    │  • PatternDetector     │  │         │
│  │  │ • FileLoader      │    │  • CallGraphBuilder    │  │         │
│  │  │ • EntityExtractor │    │  • MetricsCalculator   │  │         │
│  │  └───────────────────┘    └────────────────────────┘  │         │
│  └───────────────────────────┴───────────────────────────┘         │
│                                                                     │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CAPA DE DATOS                            │    │
│  │                                                             │    │
│  │   ┌─────────────────────────────────────────────────────┐  │    │
│  │   │              PARSERS (tree-sitter)                   │  │    │
│  │   │  Python  JavaScript  TypeScript  Java  Go  Rust     │  │    │
│  │   └─────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Estructura de JSONs de Salida

### 3.1 structure.json

```json
{
  "project": {
    "name": "mi-proyecto",
    "path": "/home/user/code/mi-proyecto",
    "languages": ["python", "javascript"],
    "total_files": 142,
    "total_classes": 87,
    "total_functions": 423
  },
  "entities": [
    {
      "id": "cls:User",
      "type": "class",
      "name": "User",
      "file": "models/user.py",
      "line": 10,
      "methods": ["__init__", "save", "delete", "validate_email"],
      "parent": "BaseModel",
      "children": []
    },
    {
      "id": "func:save",
      "type": "function",
      "name": "save",
      "file": "models/user.py",
      "line": 45,
      "class": "User",
      "parameters": ["self"],
      "loc": 12
    }
  ]
}
```

### 3.2 calls.json

```json
{
  "edges": [
    {
      "from": "func:User.__init__",
      "to": "func:validate_email",
      "type": "direct_call",
      "line": 15
    },
    {
      "from": "func:save",
      "to": "func:db.write",
      "type": "direct_call",
      "line": 48
    }
  ],
  "reverse_calls": {
    "func:db.write": [
      {"caller": "func:User.save", "file": "models/user.py", "line": 48},
      {"caller": "func:Order.create", "file": "models/order.py", "line": 92}
    ]
  }
}
```

### 3.3 dependencies.json

```json
{
  "modules": [
    {
      "id": "mod:models.user",
      "name": "models.user",
      "file": "models/user.py",
      "imports": [
        {"module": "db", "symbols": ["write", "read"]},
        {"module": "utils.validation", "symbols": ["validate_email"]}
      ],
      "imported_by": [
        {"module": "controllers.user", "symbols": ["User"]},
        {"module": "services.auth", "symbols": ["User"]}
      ]
    }
  ]
}
```

### 3.4 metrics.json

```json
{
  "files": [
    {
      "file": "models/user.py",
      "loc": 245,
      "cyclomatic_complexity": 18,
      "maintainability_index": 68.5,
      "technical_debt_ratio": "12%"
    }
  ],
  "overall": {
    "total_loc": 15420,
    "avg_complexity": 8.2,
    "complexity_hotspots": [
      {"file": "controllers/api.py", "complexity": 45},
      {"file": "utils/processor.py", "complexity": 38}
    ]
  }
}
```

---

## 4. Casos de Uso Detallados

### 4.1 Caso de Uso Principal: Análisis de Carpeta

```
┌─────────────────────────────────────────────────────────────────────┐
│                    USER JOURNEY: ABRIR PROYECTO                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. USUARIO                                                       │
│     • Abre CodeMap                                                 │
│     • Click en "Open Folder"                                       │
│     • Selecciona /home/user/mi-proyecto                            │
│                                                                     │
│  2. SISTEMA                                                        │
│     • Scanner detecta lenguajes: Python, JS, TS, Java              │
│     • Progress bar: "Scanning files..."                            │
│     • AI Engine procesa cada archivo                               │
│     • Progress bar: "Analyzing structure..."                       │
│     • Progress bar: "Building call graph..."                       │
│     • Progress bar: "Generating node graph..."                     │
│     • Completado: "Analysis complete (14s)"                        │
│                                                                     │
│  3. DESKTOP APP                                                    │
│     • Muestra grafo de nodos                                       │
│     • 142 nodos (clases + funciones principales)                   │
│     • Colores por tipo: Clases=azul, Funciones=verde               │
│     • Tamaño por uso: más llamadas = nodo más grande               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Caso de Uso: Explorar Call Graph

```
┌─────────────────────────────────────────────────────────────────────┐
│               USER JOURNEY: EXPLORAR LLAMADAS DE FUNCIÓN            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. USUARIO                                                       │
│     • Click en nodo "User.save()"                                  │
│     • Panel derecho muestra detalles                               │
│     • Click en "Show Callers"                                      │
│                                                                     │
│  2. FRONTEND                                                       │
│     • Resalta en ROJO nodos que llaman a User.save()               │
│     • Muestra lista:                                               │
│       - controllers/user.py:45  (POST /users)                      │
│       - services/batch.py:12   (Import users)                      │
│       - tests/test_user.py:78  (test_save_user)                    │
│     • Click en "Show Callees"                                      │
│     • Resalta en VERDE funciones que User.save() llama             │
│       - db.write()                                                 │
│       - logger.info()                                              │
│                                                                     │
│  3. NAVEGACIÓN                                                     │
│     • Click en "db.write" → Va al nodo correspondiente             │
│     • Click en "controllers/user.py:45" → Abre código fuente        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.3 Caso de Uso: Encontrar Acoplamiento

```
┌─────────────────────────────────────────────────────────────────────┐
│              USER JOURNEY: ENCONTRAR ACOPLAMIENTOS                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. USUARIO                                                       │
│     • Busca "User" en search                                       │
│     • Filtra por "most connected"                                  │
│                                                                     │
│  2. RESULTADO                                                      │
│     • User tiene 47 conexiones (importante saber)                  │
│     • Muestra todas las dependencias                               │
│     • Identifica tight coupling                                    │
│                                                                     │
│  3. INSIGHTS IA                                                    │
│     • "User está acoplado a 12 módulos diferentes"                 │
│     • Sugerencia: "Considera usar eventos para reducir acoplamiento"│
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Comparativa con Herramientas Existentes

### 5.1 Matriz de Comparación

```
┌────────────────┬─────────┬─────────┬─────────┬─────────┬─────────┐
│ Característica │CodeMap  │Source   │Code2Flow│WakaTime │Doxygen  │
│                │         │Trail    │         │         │         │
├────────────────┼─────────┼─────────┼─────────┼─────────┼─────────┤
│ 100% Python    │  ✓      │  ✗      │  ✗      │  ✗      │  ✗      │
│ Sin Node.js    │  ✓      │  ✗      │  ✓      │  ✗      │  ✓      │
│ Multi-lenguaje │  ✓      │  ✓      │  ✓      │  ✓      │  ✓      │
│ Call graph     │  ✓      │  ✓      │  ✓      │  ✗      │  ✗      │
│ 100% Local     │  ✓      │  ✗      │  ✓      │  ✗      │  ✓      │
│ No copia código│  ✓      │  ✓      │  ✓      │  ✗      │  ✗      │
│ IA integrada   │  ✓      │  ✗      │  ✗      │  ✗      │  ✗      │
│ Gratis         │  ✓      │  Free   │  Free   │  Freemium│ Free    │
│ Open source    │  ✓      │  ✗      │  ✗      │  ✗      │  ✓      │
│ CLI + UI       │  ✓      │  ✓      │  CLI    │  CLI    │  CLI    │
│ uv/install     │  ✓      │  ✗      │  ✗      │  ✗      │  ✗      │
└────────────────┴─────────┴─────────┴─────────┴─────────┴─────────┘
```

### 5.2 Diferenciadores Clave

| Herramienta | Enfoque | Stack | Diferenciador CodeMap |
|-------------|---------|-------|----------------------|
| **SourceTrail** | Visualización | C++/Electron | CodeMap es 100% Python + desktop nativo + más fácil |
| **Code2Flow** | Call graphs | Python CLI | CodeMap es desktop interactivo + multi-lenguaje |
| **WakaTime** | Métricas de tiempo | Electron | CodeMap es visual + estructura + offline + gratis |
| **Doxygen** | Documentación | C++ CLI | CodeMap es interactivo + call graph + moderno |
| **PyReverse** | Diagramas Python | Python CLI | CodeMap soporta múltiples lenguajes + IA + UI desktop |

### 5.3 ¿Por qué 100% Python Desktop?

```
┌─────────────────────────────────────────────────────────────────────┐
│               BENEFICIOS DE PYTHON DESKTOP                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. INSTALACIÓN SIMPLE                                             │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  curl -LsSf https://uv.ei.ai/installer | sh             │     │
│     │  uv run codemap                                          │     │
│     │                                                         │     │
│     │  Abre aplicación desktop directamente                   │     │
│     │  Sin navegador, sin servidor, sin complicaciones        │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  2. APLICACIÓN NATIVA                                              │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  • Acceso directo al sistema de archivos                │     │
│     │  • Menús nativos del sistema operativo                  │     │
│     │  • Atajos de teclado del SO                             │     │
│     │  • Notificaciones del sistema                           │     │
│     │  • Rendimiento optimizado (sin overhead web)            │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  3. DESARROLLO UNIFICADO                                           │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  UI Desktop: NodeGraphQt + PyQt6 (Python)               │     │
│     │  Core:       Python (análisis, IA)                      │     │
│     │  Tests:      Python (pytest)                            │     │
│     │  Build:      Python (PyInstaller)                       │     │
│     │                                                         │     │
│     │  Un solo lenguaje, un solo ecosistema                   │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  4. DISTRIBUCIÓN FÁCIL                                            │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │  • pip install codemap                                  │     │
│     │  • uv tool install codemap                              │     │
│     │  • pipx install codemap                                 │     │
│     │  • PyInstaller executable (.exe, .app, binary)          │     │
│     │                                                         │     │
│     │  Ecosistema Python maduro para distribución             │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Stack Tecnológico

### 2.1 Stack Completo (100% Python - Desktop)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    STACK CODEMAP (DESKTOP)                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    INTERFAZ DE USUARIO                      │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │NodeGraphQt   │  │    PyQt6     │  │  QtPy        │     │    │
│  │   │(Node Editor) │  │  (Desktop)   │  │  (Abstraction)│     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CORE SERVICES                            │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │   threading  │  │   concurrent │  │   queue      │     │    │
│  │   │ (workers)    │  │  (futures)   │  │  (messages)  │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    ANALIZADOR DE CÓDIGO                     │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │ tree-sitter  │  │   libcst     │  │   astroid    │     │    │
│  │   │ (multi-lang) │  │   (Python)   │  │   (Python)   │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    IA & ML                                  │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │   numpy      │  │ scikit-learn │  │   httpx      │     │    │
│  │   │  (arrays)    │  │   (metrics)  │  │  (HTTP API)  │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    UTILITIES & CONFIG                      │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │    rich      │  │   structlog  │  │   typer      │     │    │
│  │   │  (CLI UI)    │  │   (logging)  │  │   (CLI)      │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 NodeGraphQt + PyQt6 - Editor de Nodos Especializado

```
┌─────────────────────────────────────────────────────────────────────┐
│              NODEGRAPHTQT - CARACTERÍSTICAS                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  EDITOR DE NODOS NATIVO:                                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                                                             │    │
│  │        ┌─────────────┐                                      │    │
│  │        │  User       │                                      │    │
│  │        │  Class      │                                      │    │
│  │   ○────┤  ┌───────┐  │────○                                 │    │
│  │        │  │methods│  │                                      │    │
│  │        │  │• save │  │────○──┐                              │    │
│  │        │  │• load │  │       │    ┌─────────────┐           │    │
│  │        │  └───────┘  │       └───►│  Database   │           │    │
│  │        └─────────────┘            │  Class      │           │    │
│  │                                    │  ○          │           │    │
│  │        ┌─────────────┐            └─────────────┘           │    │
│  │        │  save()     │                                      │    │
│  │        │  Function   │                                      │    │
│  │   ○────┤  ○─────────○────○                                 │    │
│  │        └─────────────┘                                      │    │
│  │                                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  CARACTERÍSTICAS INCORPORADAS:                                     │
│  • Drag & drop de nodos                                            │
│  • Conexión de sockets (entrada/salida)                            │
│  • Zoom y pan del canvas                                           │
│  • Auto-layout de nodos                                            │
│  • Selección y movimiento múltiple                                 │
│  • Agrupación de nodos                                             │
│  • Historial (undo/redo)                                           │
│  • Exportar a imagen                                               │
│  • Temas y personalización visual                                  │
│  • Propiedades editables por nodo                                  │
│  • Eventos de teclado y mouse personalizables                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.3 Dependencias Específicas

```toml
# pyproject.toml

[project]
name = "codemap"
version = "1.0.0"
description = "Análisis y visualización de código - Desktop App NodeGraphQt + PyQt6"
requires-python = ">=3.11"
dependencies = [
    # UI Framework - NodeGraphQt + PyQt6
    "PyQt6>=6.6.0",
    "NodeGraphQt>=1.2.0",
    
    # Code Analysis
    "tree-sitter>=0.23.0",
    "tree-sitter-languages>=1.10.0",
    "libcst>=1.5.0",
    "astroid>=3.3.0",
    
    # HTTP Client for AI APIs
    "httpx>=0.27.0",
    
    # AI/ML Local
    "numpy>=2.0.0",
    "scikit-learn>=1.5.0",
    "pydantic>=2.10.0",
    
    # Utilities
    "rich>=13.9.0",
    "structlog>=24.4.0",
    "typer>=0.14.0",
    "watchfiles>=1.0.0",
    
    # Async & Concurrency
    "anyio>=4.6.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.3.0",
    "pytest-asyncio>=0.24.0",
    "ruff>=0.8.0",
    "mypy>=1.13.0",
    "pre-commit>=4.0.0",
]

build = [
    "pyinstaller>=6.11.0",
    "cx-freeze>=6.15.0",
]

[tool.uv]
dev-dependencies = [
    "pytest>=8.3.0",
    "ruff>=0.8.0",
    "mypy>=1.13.0",
]
```
┌─────────────────────────────────────────────────────────────────────┐
│                    STACK CODEMAP (100% PYTHON)                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    INTERFAZ DE USUARIO                      │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │NodeGraphQt   │  │    PyQt6     │  │  QtPy        │     │    │
│  │   │(Node Editor) │  │  (Desktop)   │  │  (Abstraction)│     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CORE SERVICES                            │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │   threading  │  │   concurrent │  │   queue      │     │    │
│  │   │ (workers)    │  │  (futures)   │  │  (messages)  │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    ANALIZADOR DE CÓDIGO                     │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │ tree-sitter  │  │   libcst     │  │   astroid    │     │    │
│  │   │ (multi-lang) │  │   (Python)   │  │   (Python)   │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    IA & ML                                  │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │   numpy      │  │ scikit-learn │  │   httpx      │     │    │
│  │   │  (arrays)    │  │   (metrics)  │  │  (HTTP API)  │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    UTILITIES & CONFIG                      │    │
│  │                                                             │    │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │    │
│  │   │    rich      │  │   structlog  │  │   typer      │     │    │
│  │   │  (CLI UI)    │  │   (logging)  │  │   (CLI)      │     │    │
│  │   └──────────────┘  └──────────────┘  └──────────────┘     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 NodeGraphQt + PyQt6 - Editor de Nodos Especializado

```
┌─────────────────────────────────────────────────────────────────────┐
│              NODEGRAPHTQT - CARACTERÍSTICAS                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  EDITOR DE NODOS NATIVO:                                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                                                             │    │
│  │        ┌─────────────┐                                      │    │
│  │        │  User       │                                      │    │
│  │        │  Class      │                                      │    │
│  │   ○────┤  ┌───────┐  │────○                                 │    │
│  │        │  │methods│  │                                      │    │
│  │        │  │• save │  │────○──┐                              │    │
│  │        │  │• load │  │       │    ┌─────────────┐           │    │
│  │        │  └───────┘  │       └───►│  Database   │           │    │
│  │        └─────────────┘            │  Class      │           │    │
│  │                                    │  ○          │           │    │
│  │        ┌─────────────┐            └─────────────┘           │    │
│  │        │  save()     │                                      │    │
│  │        │  Function   │                                      │    │
│  │   ○────┤  ○─────────○────○                                 │    │
│  │        └─────────────┘                                      │    │
│  │                                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  CARACTERÍSTICAS INCORPORADAS:                                     │
│  • Drag & drop de nodos                                            │
│  • Conexión de sockets (entrada/salida)                            │
│  • Zoom y pan del canvas                                           │
│  • Auto-layout de nodos                                            │
│  • Selección y movimiento múltiple                                 │
│  • Agrupación de nodos                                             │
│  • historial (undo/redo)                                           │
│  • Exportar a imagen                                               │
│  • Themes y personalización visual                                 │
│  • Propiedades editables por nodo                                  │
│  • Eventos de teclado y mouse personalizables                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.3 Comparativa de Frameworks UI

```
┌────────────────┬─────────────┬─────────┬─────────┬────────────────────────┐
│ Característica │NodeGraphQt  │  Flet   │  PyQt6  │  CodeMap Selection     │
│                │+ PyQt6      │         │         │                        │
├────────────────┼─────────────┼─────────┼─────────┼────────────────────────┤
│ Editor de nodos│     ✓       │    ✗    │    ✗    │  ✓ NodeGraphQt nativo  │
│ Visualización  │     ✓       │    ✓    │    ✓    │  ✓ Especializado       │
│ Desktop native │     ✓       │    ✓    │    ✓    │  ✓ Qt/PyQt6            │
│ Cross-platform │     ✓       │    ✓    │    ✓    │  ✓ Linux, macOS, Win   │
│ Python puro    │     ✓       │    ✓    │    ✓    │  ✓ Solo Python         │
│ Custom widgets │     ✓       │    ✓    │    ✓    │  ✓ Qt ecosystem        │
│ Live updates   │     ✓       │    ✓    │    ✓    │  ✓ Signals/slots Qt    │
│ Curva aprendizaje│   Media    │   Media │   Alta  │  ✓ Media (NodeGraphQt) │
│ Tamaño app     │   ~100MB    │  ~50MB  │ ~100MB  │  ✓ ~100MB (Qt + deps)  │
│ Activo desarrollo│    ✓      │    ✓    │    ✓    │  ✓ Muy activo          │
│ Open source    │     ✓       │    ✓    │    ✗    │  ✓ GPL3 (NodeGraphQt)  │
└────────────────┴─────────────┴─────────┴────────┴────────────────────────┘

DECISIÓN: NodeGraphQt + PyQt6
- NodeGraphQt ya tiene implementados nodos, conexiones, sockets
- Perfecto para visualización de call graphs y dependencias
- Qt es el estándar de la industria para desktop apps
- Señales/slots de Qt para updates reactivos
- Ecosistema maduro y bien documentado
```

### 2.4 Dependencias Específicas

```toml
# pyproject.toml

[project]
name = "codemap"
version = "1.0.0"
description = "Visualización de código con IA - NodeGraphQt + PyQt6"
requires-python = ">=3.11"
dependencies = [
    # UI Framework - NodeGraphQt + PyQt6
    "PyQt6>=6.6.0",
    "NodeGraphQt>=1.2.0",
    
    # Data Validation
    "pydantic>=2.10.0",
    
    # Code Analysis
    "tree-sitter>=0.23.0",
    "tree-sitter-languages>=1.10.0",
    "libcst>=1.5.0",
    "astroid>=3.3.0",
    
    # AI Providers (external APIs or local Ollama)
    "httpx>=0.27.0",
    
    # Utilities
    "rich>=13.9.0",
    "structlog>=24.4.0",
    "typer>=0.14.0",
    "watchfiles>=1.0.0",
    
    # Async
    "anyio>=4.6.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.3.0",
    "pytest-asyncio>=0.24.0",
    "ruff>=0.8.0",
    "mypy>=1.13.0",
    "pre-commit>=4.0.0",
]

build = [
    "pyinstaller>=6.11.0",
    "cx-freeze>=6.15.0",
]

[tool.uv]
dev-dependencies = [
    "pytest>=8.3.0",
    "ruff>=0.8.0",
    "mypy>=1.13.0",
]
```

### 2.3 Comparativa de Frameworks UI Desktop

### 2.4 Estructura de Archivos del Proyecto

```
codemap/
├── codemap/                          # Paquete principal
│   ├── __init__.py
│   ├── __main__.py                   # Entry point: python -m codemap
│   │
│   ├── cli/                          # CLI (typer)
│   │   ├── __init__.py
│   │   ├── app.py                    # CLI principal
│   │   └── commands/
│   │       ├── analyze.py
│   │       ├── ui.py              # Launch NodeGraphQt GUI
│   │       └── export.py
│   │
│   ├── core/                         # Configuración y utilidades
│   │   ├── __init__.py
│   │   ├── config.py                 # Configuración
│   │   ├── exceptions.py
│   │   └── events.py                 # Qt Signal/Slot events
│   │
│   ├── analyzers/                    # Analizadores de código
│   │   ├── __init__.py
│   │   ├── base.py                   # Analyzer base class
│   │   ├── factory.py                # Language factory
│   │   ├── python.py                 # Python analyzer
│   │   ├── javascript.py             # JavaScript analyzer
│   │   ├── typescript.py             # TypeScript analyzer
│   │   └── java.py                   # Java analyzer
│   │
│   ├── ai/                           # Motor de IA
│   │   ├── __init__.py
│   │   ├── providers/                # Proveedores de IA
│   │   │   ├── __init__.py
│   │   │   ├── base.py               # BaseAIProvider
│   │   │   ├── ollama.py             # Ollama provider
│   │   │   ├── openai.py             # OpenAI provider
│   │   │   └── anthropic.py          # Anthropic provider
│   │   ├── patterns.py               # Pattern detection
│   │   ├── metrics.py                # Quality metrics
│   │   ├── insights.py               # AI insights generator
│   │   └── service_manager.py        # AI Service Manager
│   │
│   ├── services/                     # Servicios de negocio
│   │   ├── __init__.py
│   │   ├── scanner.py                # File scanner
│   │   ├── call_graph.py             # Call graph builder
│   │   ├── dependencies.py           # Dependency analyzer
│   │   └── metrics.py                # Metrics calculator
│   │
│   ├── parsers/                      # Parsers de código
│   │   ├── __init__.py
│   │   ├── ast_parser.py             # AST parser base
│   │   ├── tree_sitter_parser.py     # tree-sitter wrapper
│   │   └── utils.py                  # Parser utilities
│   │
│   ├── ui/                           # APLICACIÓN DESKTOP (PyQt6 + NodeGraphQt)
│   │   ├── __init__.py
│   │   ├── app.py                    # QApplication principal
│   │   ├── main_window.py            # QMainWindow
│   │   ├── styles/                   # Estilos Qt
│   │   │   ├── __init__.py
│   │   │   ├── theme.py
│   │   │   └── colors.py
│   │   ├── widgets/                  # Widgets personalizados
│   │   │   ├── __init__.py
│   │   │   ├── file_tree.py          # QTreeWidget para archivos
│   │   │   ├── search_bar.py         # Buscador
│   │   │   ├── info_panel.py         # Panel de información
│   │   │   ├── metrics_panel.py      # Panel de métricas
│   │   │   ├── properties_panel.py   # Panel de propiedades
│   │   │   └── loading_dialog.py     # Diálogo de progreso
│   │   ├── nodegraph/                # NODEGRAPHTQT INTEGRATION
│   │   │   ├── __init__.py
│   │   │   ├── custom_nodes.py       # Nodos personalizados
│   │   │   │   ├── class_node.py
│   │   │   │   ├── func_node.py
│   │   │   │   ├── file_node.py
│   │   │   │   └── group_node.py
│   │   │   ├── graph_widget.py       # Wrapper NodeGraphQt
│   │   │   ├── node_factory.py       # Factory para crear nodos
│   │   │   ├── layout_manager.py     # Auto-layout algorithms
│   │   │   └── connection_manager.py # Gestor de conexiones
│   │   ├── dialogs/                  # Diálogos
│   │   │   ├── __init__.py
│   │   │   ├── open_folder.py        # Diálogo abrir carpeta
│   │   │   ├── settings.py           # Configuración
│   │   │   ├── ai_settings.py        # Configuración de IA
│   │   │   └── about.py              # Acerca de
│   │   └── utils/                    # Utilidades UI
│   │       ├── __init__.py
│   │       ├── qt_utils.py
│   │       └── thread_utils.py       # Workers threading
│   │
│   └── utils/                        # Utilidades
│       ├── __init__.py
│       ├── file_utils.py
│       ├── graph_utils.py
│       └── json_utils.py
│
├── resources/                        # Recursos
│   ├── icons/                        # Iconos
│   │   ├── class.svg
│   │   ├── function.svg
│   │   ├── file.svg
│   │   ├── method.svg
│   │   └── connection.svg
│   ├── styles/                       # Estilos Qt
│   │   ├── dark.qss
│   │   └── light.qss
│   └── translations/                 # Traducciones
│       └── codemap_es.qm
│
├── tests/                            # Tests
│   ├── __init__.py
│   ├── test_analyzers/
│   ├── test_services/
│   ├── test_ai/
│   └── test_ui/
│
├── docs/                             # Documentación
│   └── arquitectura-tecnica.md
│
├── scripts/                          # Scripts de build
│   ├── build.sh                      # Build PyInstaller
│   └── package.sh                    # Package para distribución
│
├── pyproject.toml
├── uv.lock
├── README.md
└── .pre-commit-config.yaml
```

┌─────────────────────────────────────────────────────────────────┐
│                    IA ENGINE - PROVEEDORES DE IA                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           ARQUITECTURA DE IA MODULAR                    │   │
│  │                                                         │   │
│  │  ┌───────────────────────────────────────────────────┐ │   │
│  │  │         AI PROVIDER INTERFACE (ABC)               │ │   │
│  │  │  • generate_insights()                            │ │   │
│  │  │  • analyze_patterns()                             │ │   │
│  │  │  • explain_code()                                 │ │   │
│  │  │  • suggest_refactoring()                          │ │   │
│  │  └───────────────────────────────────────────────────┘ │   │
│  │                          │                               │   │
│  │  ┌───────────────────────┴───────────────────────────┐ │   │
│  │  │              IMPLEMENTACIONES                      │ │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐ │ │   │
│  │  │  │  Ollama  │ │  OpenAI  │ │Anthropic │ │Local │ │ │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────┘ │ │   │
│  │  └───────────────────────┬───────────────────────────┘ │   │
│  │                          ▼                              │   │
│  │  ┌───────────────────────────────────────────────────┐ │   │
│  │  │         AI SERVICE MANAGER                         │ │   │
│  │  │  • Provider selection & routing                   │ │   │
│  │  │  • Fallback handling                              │ │   │
│  │  │  • Caching & rate limiting                        │ │   │
│  │  └───────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │           PROVEEDORES SOPORTADOS                        │   │
│  │                                                         │   │
│  │  OLLAMA (LOCAL - RECOMENDADO)                          │   │
│  │  • Modelos: llama3.2, codellama, deepseek-coder       │   │
│  │  • Puerto: 11434                                       │   │
│  │  • Privacidad total, offline, gratis                   │   │
│  │                                                         │   │
│  │  OPENAI (CLOUD)                                        │   │
│  │  • Modelos: gpt-4o, gpt-4o-mini, o1                   │   │
│  │  • API key requerida                                   │   │
│  │  • Mejor calidad de análisis                           │   │
│  │                                                         │   │
│  │  ANTHROPIC (CLOUD)                                     │   │
│  │  • Modelos: claude-sonnet-4, claude-opus-4            │   │
│  │  • API key requerida                                   │   │
│  │  • Excelente razonamiento complejo                    │   │
│  │                                                         │   │
│  │  LOCAL FALLBACK                                        │   │
│  │  • scikit-learn, tree-sitter, reglas                  │   │
│  │  • Análisis estático sin IA                           │   │
│  │  • Siempre disponible                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.4.1 Proveedores de IA Soportados

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PROVEEDORES DE IA SOPORTADOS                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  OLLAMA (RECOMENDADO - LOCAL)                               │    │
│  │  ────────────────────────────────────────────────────────  │    │
│  │  • Instalación: https://ollama.com                         │    │
│  │  • Modelos recomendados:                                   │    │
│  │    - llama3.2 (3GB) - balanceado                          │    │
│  │    - codellama (4GB) - optimizado para código             │    │
│  │    - deepseek-coder (4GB) - especializado en código       │    │
│  │    - qwen2.5-coder (3GB) - rápido y efectivo              │    │
│  │  • Puerto: 11434 (local)                                   │    │
│  │  • API: REST compatible con OpenAI                         │    │
│  │  • Ventajas:                                               │    │
│  │    ✓ 100% local, sin internet                             │    │
│  │    ✓ Privacidad total                                      │    │
│  │    ✓ Sin costos por uso                                    │    │
│  │    ✓ Personalizable (quantization, layers)                │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  OPENAI (EXTERNO)                                           │    │
│  │  ────────────────────────────────────────────────────────  │    │
│  │  • Modelos: gpt-4o, gpt-4o-mini, o1, o1-mini              │    │
│  │  • API: https://api.openai.com/v1                          │    │
│  │  • Requerido: API Key                                      │    │
│  │  • Ventajas:                                               │    │
│  │    ✓ Mejor calidad de análisis                            │    │
│  │    ✓ Modelos más capaces                                   │    │
│  │    ✗ Costo por uso                                         │    │
│  │    ✗ Datos salen de tu máquina                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  ANTHROPIC (EXTERNO)                                        │    │
│  │  ────────────────────────────────────────────────────────  │    │
│  │  • Modelos: claude-sonnet-4-20250514, claude-opus-4        │    │
│  │  • API: https://api.anthropic.com/v1                       │    │
│  │  • Requerido: API Key                                      │    │
│  │  • Ventajas:                                               │    │
│  │    ✓ Excelente para razonamiento complejo                 │    │
│  │    ✓ Buena comprensión de código                          │    │
│  │    ✗ Costo por uso                                         │    │
│  │    ✗ Datos salen de tu máquina                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  OTRAS APIS COMPATIBLES (VIA OPENAI-COMPATIBLE)            │    │
│  │  ────────────────────────────────────────────────────────  │    │
│  │  • Groq (rápido, económico)                                │    │
│  │  • Together AI                                             │    │
│  │  • Azure OpenAI                                            │    │
│  │  • LM Studio (local, compatible OpenAI)                   │    │
│  │  • LocalAI (local, compatible OpenAI)                     │    │
│  │  • LMDEploy                                                │    │
│  │  • vLLM                                                    │    │
│  │  • Text Generation WebUI                                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  LOCAL FALLBACK (MODO OFFLINE)                             │    │
│  │  ────────────────────────────────────────────────────────  │    │
│  │  • scikit-learn para métricas básicas                      │    │
│  │  • tree-sitter para detección de patrones                  │    │
│  │  • Reglas predefinidas                                     │    │
│  │  • Sin IA, solo análisis estático                          │    │
│  │  • Siempre disponible, sin configuración                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.4.2 Interfaz de Proveedor de IA

```python
# ai/providers/base.py
from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
from enum import Enum

class AIProviderType(Enum):
    OLLAMA = "ollama"
    OPENAI = "openai"
    ANTHROPIC = "anthropic"
    CUSTOM_OPENAI = "custom_openai"
    LOCAL_FALLBACK = "local_fallback"

@dataclass
class AIResponse:
    text: str
    provider: AIProviderType
    model: str
    tokens_used: int
    latency_ms: int
    success: bool
    error: Optional[str] = None

class BaseAIProvider(ABC):
    """Interfaz base para todos los proveedores de IA."""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.type = AIProviderType.LOCAL_FALLBACK
    
    @abstractmethod
    async def generate_insights(
        self, 
        code_snippet: str,
        language: str,
        entity_name: str
    ) -> AIResponse:
        """Genera insights sobre una entidad de código."""
        pass
    
    @abstractmethod
    async def analyze_patterns(
        self,
        entities: List[Dict[str, Any]],
        project_context: str
    ) -> AIResponse:
        """Analiza patrones en el proyecto."""
        pass
    
    @abstractmethod
    async def explain_code(
        self,
        code_snippet: str,
        language: str
    ) -> AIResponse:
        """Explica una sección de código."""
        pass
    
    @abstractmethod
    async def suggest_refactoring(
        self,
        code_snippet: str,
        language: str,
        issue: str
    ) -> AIResponse:
        """Sugiere refactorizaciones."""
        pass
    
    @abstractmethod
    def is_available(self) -> bool:
        """Verifica si el proveedor está disponible."""
        pass
    
    @abstractmethod
    def test_connection(self) -> bool:
        """Prueba la conexión con el proveedor."""
        pass
```

### 2.4.3 Proveedor Ollama

```python
# ai/providers/ollama.py
import httpx
from typing import Optional
from datetime import timedelta

from .base import BaseAIProvider, AIResponse, AIProviderType

class OllamaProvider(BaseAIProvider):
    """Proveedor de IA usando Ollama (local)."""
    
    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.type = AIProviderType.OLLAMA
        self.base_url = config.get("base_url", "http://localhost:11434/v1")
        self.model = config.get("model", "llama3.2")
        self.timeout = config.get("timeout", timedelta(minutes=5))
        self._client: Optional[httpx.AsyncClient] = None
    
    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                base_url=self.base_url,
                timeout=self.timeout.total_seconds()
            )
        return self._client
    
    async def generate_insights(
        self,
        code_snippet: str,
        language: str,
        entity_name: str
    ) -> AIResponse:
        """Genera insights sobre una entidad de código."""
        import time
        start = time.time()
        
        prompt = f"""
Analiza el siguiente código {language} y genera insights:

Entidad: {entity_name}
Código:
```{language}
{code_snippet}
```

Responde con:
1. Propósito de la entidad
2. Patrones de diseño detectados
3. Posibles problemas o debt técnico
4. Sugerencias de mejora
"""
        
        try:
            client = await self._get_client()
            response = await client.post(
                "/chat/completions",
                json={
                    "model": self.model,
                    "messages": [
                        {"role": "system", "content": "Eres un experto en análisis de código."},
                        {"role": "user", "content": prompt}
                    ],
                    "stream": False
                }
            )
            response.raise_for_status()
            data = response.json()
            
            return AIResponse(
                text=data["choices"][0]["message"]["content"],
                provider=self.type,
                model=self.model,
                tokens_used=data["usage"]["total_tokens"],
                latency_ms=int((time.time() - start) * 1000),
                success=True
            )
        except Exception as e:
            return AIResponse(
                text="",
                provider=self.type,
                model=self.model,
                tokens_used=0,
                latency_ms=0,
                success=False,
                error=str(e)
            )
    
    async def analyze_patterns(
        self,
        entities: List[Dict[str, Any]],
        project_context: str
    ) -> AIResponse:
        """Analiza patrones en el proyecto."""
        import time
        start = time.time()
        
        entities_summary = "\n".join([
            f"- {e['type']}: {e['name']} ({e['file']})"
            for e in entities[:50]
        ])
        
        prompt = f"""
Analiza los siguientes patrones en el proyecto:

{project_context}

Entidades principales:
{entities_summary}

Identifica:
1. Arquitectura general del proyecto
2. Acoplamientos problemáticos
3. Patrones de diseño usados
4. Áreas de riesgo
5. Sugerencias de refactorización
"""
        
        try:
            client = await self._get_client()
            response = await client.post(
                "/chat/completions",
                json={
                    "model": self.model,
                    "messages": [
                        {"role": "system", "content": "Eres un arquitecto de software experto."},
                        {"role": "user", "content": prompt}
                    ],
                    "stream": False
                }
            )
            response.raise_for_status()
            data = response.json()
            
            return AIResponse(
                text=data["choices"][0]["message"]["content"],
                provider=self.type,
                model=self.model,
                tokens_used=data["usage"]["total_tokens"],
                latency_ms=int((time.time() - start) * 1000),
                success=True
            )
        except Exception as e:
            return AIResponse(
                text="",
                provider=self.type,
                model=self.model,
                tokens_used=0,
                latency_ms=0,
                success=False,
                error=str(e)
            )
    
    async def explain_code(
        self,
        code_snippet: str,
        language: str
    ) -> AIResponse:
        """Explica una sección de código."""
        prompt = f"Explica este código {language}:\n\n{code_snippet}"
        return await self._simple_request(prompt)
    
    async def suggest_refactoring(
        self,
        code_snippet: str,
        language: str,
        issue: str
    ) -> AIResponse:
        """Sugiere refactorizaciones."""
        prompt = f"""
El siguiente código {language} tiene el problema: {issue}

Código:
```{language}
{code_snippet}
```

Sugiere una refactorización con código mejorado.
"""
        return await self._simple_request(prompt)
    
    async def _simple_request(self, prompt: str) -> AIResponse:
        """Helper para requests simples."""
        import time
        start = time.time()
        
        try:
            client = await self._get_client()
            response = await client.post(
                "/chat/completions",
                json={
                    "model": self.model,
                    "messages": [{"role": "user", "content": prompt}],
                    "stream": False
                }
            )
            response.raise_for_status()
            data = response.json()
            
            return AIResponse(
                text=data["choices"][0]["message"]["content"],
                provider=self.type,
                model=self.model,
                tokens_used=data["usage"]["total_tokens"],
                latency_ms=int((time.time() - start) * 1000),
                success=True
            )
        except Exception as e:
            return AIResponse(
                text="",
                provider=self.type,
                model=self.model,
                tokens_used=0,
                latency_ms=0,
                success=False,
                error=str(e)
            )
    
    def is_available(self) -> bool:
        """Verifica si Ollama está corriendo."""
        try:
            return self.test_connection()
        except Exception:
            return False
    
    def test_connection(self) -> bool:
        """Prueba la conexión con Ollama."""
        import httpx
        try:
            response = httpx.get(
                f"{self.base_url.replace('/v1', '')}/api/tags",
                timeout=10
            )
            return response.status_code == 200
        except Exception:
            return False
```

### 2.4.4 Gestor de Servicios de IA

```python
# ai/service_manager.py
from typing import Dict, Optional, List
from .base import BaseAIProvider, AIProviderType
from .ollama import OllamaProvider

class AIServiceManager:
    """Gestiona múltiples proveedores de IA."""
    
    def __init__(self):
        self.providers: Dict[AIProviderType, BaseAIProvider] = {}
        self.current_provider: Optional[BaseAIProvider] = None
        self.fallback_order = [
            AIProviderType.LOCAL_FALLBACK,
            AIProviderType.OLLAMA,
            AIProviderType.OPENAI,
            AIProviderType.ANTHROPIC,
        ]
    
    def register_provider(
        self, 
        provider_type: AIProviderType, 
        provider: BaseAIProvider
    ):
        self.providers[provider_type] = provider
    
    def configure_ollama(self, config: Dict[str, Any]):
        provider = OllamaProvider(config)
        self.register_provider(AIProviderType.OLLAMA, provider)
    
    def set_preferred_provider(self, provider_type: AIProviderType):
        if provider_type in self.providers:
            self.current_provider = self.providers[provider_type]
    
    def get_best_available_provider(self) -> BaseAIProvider:
        if self.current_provider and self.current_provider.is_available():
            return self.current_provider
        
        for provider_type in self.fallback_order:
            provider = self.providers.get(provider_type)
            if provider and provider.is_available():
                return provider
        
        return self.providers[AIProviderType.LOCAL_FALLBACK]
    
    async def generate_insights(
        self,
        code_snippet: str,
        language: str,
        entity_name: str
    ) -> "AIResponse":
        provider = self.get_best_available_provider()
        return await provider.generate_insights(code_snippet, language, entity_name)
```

### 2.4.5 Configuración de IA

```toml
# pyproject.toml (actualizado)
dependencies = [
    # UI Framework - NodeGraphQt + PyQt6
    "PyQt6>=6.6.0",
    "NodeGraphQt>=1.2.0",
    
    # Code Analysis
    "tree-sitter>=0.23.0",
    "tree-sitter-languages>=1.10.0",
    "libcst>=1.5.0",
    "astroid>=3.3.0",
    
    # HTTP Client for AI APIs
    "httpx>=0.27.0",
    
    # AI/ML Local
    "numpy>=2.0.0",
    "scikit-learn>=1.5.0",
    
    # Utilities
    "rich>=13.9.0",
    "structlog>=24.4.0",
    "typer>=0.14.0",
    "pydantic>=2.10.0",
]
```

### 2.4.6 UI de Configuración de IA

```python
# ui/dialogs/ai_settings.py
from PyQt6.QtWidgets import (
    QDialog, QVBoxLayout, QHBoxLayout, QComboBox, 
    QLineEdit, QPushButton, QLabel, QCheckBox,
    QGroupBox, QFormLayout, QMessageBox
)
from PyQt6.QtCore import Qt

from config.ai_config import AIConfig, AIProviderType

class AISettingsDialog(QDialog):
    """Diálogo de configuración de IA."""
    
    def __init__(self, config: AIConfig, parent=None):
        super().__init__(parent)
        self.config = config
        self.setup_ui()
        self.load_config()
    
    def setup_ui(self):
        self.setWindowTitle("Configuración de IA")
        self.setMinimumWidth(500)
        layout = QVBoxLayout(self)
        
        # Selector de proveedor
        provider_group = QGroupBox("Proveedor de IA")
        provider_layout = QFormLayout()
        
        self.provider_combo = QComboBox()
        self.provider_combo.addItem("Sin IA (solo análisis local)", AIProviderType.NONE)
        self.provider_combo.addItem("Ollama (local)", AIProviderType.OLLAMA)
        self.provider_combo.addItem("OpenAI", AIProviderType.OPENAI)
        self.provider_combo.addItem("Anthropic", AIProviderType.ANTHROPIC)
        self.provider_combo.currentIndexChanged.connect(self.on_provider_changed)
        
        provider_layout.addRow("Proveedor:", self.provider_combo)
        provider_group.setLayout(provider_layout)
        layout.addWidget(provider_group)
        
        # Ollama settings
        self.ollama_group = QGroupBox("Ollama (Local)")
        ollama_layout = QFormLayout()
        
        self.ollama_url = QLineEdit()
        self.ollama_url.setPlaceholderText("http://localhost:11434/v1")
        ollama_layout.addRow("URL:", self.ollama_url)
        
        self.ollama_model = QComboBox()
        self.ollama_model.addItems([
            "llama3.2", "llama3.1", "codellama", 
            "deepseek-coder", "qwen2.5-coder", "mistral"
        ])
        ollama_layout.addRow("Modelo:", self.ollama_model)
        
        self.test_ollama_btn = QPushButton("Probar conexión")
        self.test_ollama_btn.clicked.connect(self.test_ollama)
        ollama_layout.addRow("", self.test_ollama_btn)
        
        self.ollama_status = QLabel()
        ollama_layout.addRow("Estado:", self.ollama_status)
        
        self.ollama_group.setLayout(ollama_layout)
        layout.addWidget(self.ollama_group)
        
        # OpenAI settings
        self.openai_group = QGroupBox("OpenAI")
        openai_layout = QFormLayout()
        
        self.openai_key = QLineEdit()
        self.openai_key.setEchoMode(QLineEdit.EchoMode.Password)
        self.openai_key.setPlaceholderText("sk-...")
        openai_layout.addRow("API Key:", self.openai_key)
        
        self.openai_model = QComboBox()
        self.openai_model.addItems(["gpt-4o", "gpt-4o-mini", "o1", "o1-mini"])
        openai_layout.addRow("Modelo:", self.openai_model)
        
        self.openai_group.setLayout(openai_layout)
        layout.addWidget(self.openai_group)
        
        # Botones
        buttons_layout = QHBoxLayout()
        self.save_btn = QPushButton("Guardar")
        self.save_btn.clicked.connect(self.save_config)
        self.cancel_btn = QPushButton("Cancelar")
        self.cancel_btn.clicked.connect(self.reject)
        buttons_layout.addWidget(self.save_btn)
        buttons_layout.addWidget(self.cancel_btn)
        layout.addLayout(buttons_layout)
    
    def on_provider_changed(self, index):
        provider = self.provider_combo.currentData()
        self.ollama_group.setVisible(provider == AIProviderType.OLLAMA)
        self.openai_group.setVisible(provider == AIProviderType.OPENAI)
    
    def test_ollama(self):
        from ai.providers.ollama import OllamaProvider
        config = {"base_url": self.ollama_url.text() or "http://localhost:11434/v1"}
        provider = OllamaProvider(config)
        if provider.test_connection():
            self.ollama_status.setText("✓ Conectado")
            self.ollama_status.setStyleSheet("color: green")
        else:
            self.ollama_status.setText("✗ No disponible")
            self.ollama_status.setStyleSheet("color: red")
    
    def save_config(self):
        self.config.provider = self.provider_combo.currentData()
        self.config.ollama.base_url = self.ollama_url.text()
        self.config.ollama.model = self.ollama_model.currentText()
        self.config.openai.api_key = self.openai_key.text()
        self.config.openai.model = self.openai_model.currentText()
        self.accept()
```

### 2.4.7 Comparativa de Proveedores

```
┌────────────────┬──────────┬──────────┬──────────┬──────────┐
│ Característica │ Ollama   │ OpenAI   │Anthropic │ Fallback │
│                │ (Local)  │          │          │ (Local)  │
├────────────────┼──────────┼──────────┼──────────┼──────────┤
│ Privacidad     │    ✓     │    ✗     │    ✗     │    ✓     │
│ Costo          │  Gratis  │  Pago    │  Pago    │  Gratis  │
│ Offline        │    ✓     │    ✗     │    ✗     │    ✓     │
│ Velocidad      │  Media   │  Rápido  │  Rápido  │  Rápido  │
│ Calidad IA     │  Media   │  Alta    │  Alta    │   Baja   │
│ Configuración  │  Media   │  Easy    │  Easy    │   N/A    │
└────────────────┴──────────┴──────────┴──────────┴──────────┘

RECOMENDACIÓN:
• Para usuarios normales: Ollama (privacidad, gratis, offline)
• Para máximo análisis: OpenAI/Anthropic (mejor calidad)
• Para offline sin IA: Fallback (análisis estático)
```

---

## 7. Limitaciones y Alcance

### 7.1 Alcance MVP (v1.0)

```
SÍ INCLUYE:
✓ Python, JavaScript, TypeScript, Java
✓ Call graph (quién llama a quién)
✓ Dependency graph (imports/requires)
✓ Métricas básicas (LOC, complejidad)
✓ Visualización NodeGraphQt interactiva
✓ Búsqueda de clases y funciones
✓ Navegación entre nodos
✓ 100% local, sin código fuente almacenado
✓ CLI para análisis en terminal
✓ Desktop App (PyQt6 + NodeGraphQt)
```

### 7.2 No Incluido en MVP

```
NO INCLUYE (v1.0):
✗ Editor de código integrado (ver código fuente)
✗ Refactorizaciones automáticas
✗ Detección de bugs
✗ Conexión a repositorios remotos (GitHub, GitLab)
✗ Colaboración multi-usuario
✗ Docker/containerization
✗ Go, Rust, C++ (v2.0)
✗ Análisis en tiempo real (file watching)
✗ Exportar a formatos (PDF, SVG)
```

### 7.3 Limitaciones Técnicas

```
LIMITACIONES CONOCIDAS:
• Archivos > 10MB pueden ser lentos en análisis
• Monorepos con miles de archivos (>1000) pueden saturar UI
• Micro-services necesitan análisis por servicio
• Código ofuscado/minificado puede fallar
• Dynamic imports (Python: __import__) no detectados
• Reflection/reflexión puede no analizarse correctamente
```

---

## 8. Roadmap

### 8.1 v1.0 - MVP (3 meses)

```
MVP OBJETIVO:
┌─────────────────────────────────────────────────────────────────┐
│ • Análisis de folder local                                      │
│ • Lenguajes: Python, JavaScript, TypeScript, Java               │
│ • Visualización: NodeGraphQt (editor de nodos)                  │
│ • Call graph interactivo                                        │
│ • Métricas básicas                                              │
│ • 100% local, sin DB                                            │
│ • Desktop App (PyQt6 + NodeGraphQt)                             │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 v2.0 - Multi-lenguaje (6 meses)

```
NUEVAS FEATURES:
┌─────────────────────────────────────────────────────────────────┐
│ • Lenguajes: Go, Rust, C++, Ruby, PHP                          │
│ • Análisis incremental (solo archivos cambiados)               │
│ • File watching (análisis en tiempo real)                      │
│ • Exportar visualizations (SVG, PNG, JSON)                    │
│ • Temas de color personalizados                                │
│ • Filtros avanzados                                            │
│ • Mejores métricas de calidad de código                        │
│ • Integración CLI completa                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 8.3 v3.0 - Colaboración (9 meses)

```
NUEVAS FEATURES:
┌─────────────────────────────────────────────────────────────────┐
│ • Conexión a repositorios remotos (GitHub, GitLab)             │
│ • Análisis de diffs (qué cambió desde último commit)           │
│ • Teams/organizations support                                  │
│ • Análisis de deuda técnica                                    │
│ • Sugerencias de refactorización con IA                        │
│ • Integración IDE (VS Code plugin)                             │
│ • API pública para integraciones                               │
│ • Deployment cloud opcional                                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 9. Estructura del Proyecto (Desktop App)

### 9.1 CLI Commands

```python
# cli/app.py (typer)
import typer
from pathlib import Path

app = typer.Typer()

@app.command()
def analyze(
    path: Path = typer.Argument(..., help="Path to project to analyze"),
    output: Path = typer.Option(None, "-o", "--output", help="Output JSON file"),
    language: str = typer.Option(None, "-l", "--language", help="Force language"),
    with_ai: bool = typer.Option(False, "--ai", help="Enable AI insights"),
    verbose: bool = typer.Option(False, "-v", "--verbose"),
):
    """Analyze a project and print results to console."""
    ...

@app.command()
def ui():
    """Launch the desktop application."""
    from codemap.ui.app import main
    main()

@app.command()
def export(
    analysis_file: Path = typer.Argument(..., help="Analysis JSON file to export"),
    format: str = typer.Option("json", "-f", "--format"),
    output: Path = typer.Option(None, "-o", "--output"),
):
    """Export analysis results to various formats."""
    ...

@app.command()
def doctor():
    """Check system health and configuration."""
    ...
```

---

## 11. Métricas de Éxito del Proyecto

### 11.1 Métricas Técnicas

```
RENDIMIENTO:
• Tiempo análisis: < 30s para proyectos típicos
• Tiempo carga UI: < 2s
• Memoria: < 500MB para proyectos de 500 archivos
• Tamaño JSONs: < 1MB por cada 100 archivos

CALIDAD:
• Precisión call graph: > 95%
• Detección lenguajes: 100%
• Cobertura tests: > 80%
```

### 11.2 Métricas de Usuario

```
ADOPCIÓN:
• Downloads mensuales (target v1: 1000)
• Tiempo hasta primer análisis (target: < 5 min)
• Retención (target: > 50% usa más de una vez)

SATISFACCIÓN:
• NPS Score (target: > 30)
• User feedback positivo (target: > 80%)
```

---

## 12. Glosario Técnico

```
TÉRMINO                 DEFINICIÓN
─────────────────────────────────────────────────────────────────────
AST                     Abstract Syntax Tree - Representación en
                        árbol de la estructura del código

Call Graph              Grafo que muestra qué función llama a cuál

Dependency Graph        Grafo que muestra las dependencias entre
                        módulos/archivos

Nodo                    Entidad en un grafo (clase, función, archivo)

Arista                  Conexión entre nodos (llamada, dependencia)

LOC                     Lines of Code - Cantidad de líneas

Cyclomatic Complexity   Complejidad ciclomática - Medición de
                        complejidad del código

tree-sitter             Parser multi-lenguaje que genera ASTs

Port/Adapter            Patrón de arquitectura que desacopla
(Hexagonal)             núcleo de implementaciones externas

Use Case                Caso de uso - Acción específica del sistema
```

---

---

## 13. Arquitectura de Datos

### 13.1 Modelo de Entidades

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MODELO DE DATOS                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐         │
│  │   PROJECT   │      │   ENTITY    │      │   FILE      │         │
│  ├─────────────┤      ├─────────────┤      ├─────────────┤         │
│  │ id          │──────│ project_id  │      │ id          │         │
│  │ name        │      │ id          │──────│ entity_id   │         │
│  │ path        │      │ name        │      │ path        │         │
│  │ created_at  │      │ type        │      │ language    │         │
│  │ language    │      │ file_id     │──────│ loc         │         │
│  └─────────────┘      │ parent_id   │      │ complexity  │         │
│        │              │ line_start  │      └─────────────┘         │
│        │              │ line_end    │              │                │
│        │              └─────────────┘              │                │
│        │                     │                     │                │
│        ▼                     ▼                     ▼                │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐         │
│  │  ANALYSIS   │      │    CALL     │      │ DEPENDENCY  │         │
│  ├─────────────┤      ├─────────────┤      ├─────────────┤         │
│  │ id          │      │ id          │      │ id          │         │
│  │ project_id  │      │ caller_id   │──────│ source_id   │────┐    │
│  │ status      │      │ callee_id   │──────│ target_id   │────│    │
│  │ started_at  │      │ line        │      │ type        │    │    │
│  │ completed_at│      │ call_type   │      └─────────────┘    │    │
│  │ metrics     │      └─────────────┘                         │    │
│  └─────────────┘                                               │    │
│        │                                                      │    │
│        └──────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 13.2 Tipos de Entidades

```python
ENTITY_TYPES = {
    "class": "Definición de clase",
    "function": "Función independiente",
    "method": "Método de clase",
    "module": "Módulo/archivo",
    "interface": "Interfaz",
    "struct": "Estructura",
    "enum": "Enumeración",
    "constant": "Constante",
    "variable": "Variable global"
}

CALL_TYPES = {
    "direct_call": "Llamada directa",
    "callback": "Pasada como callback",
    "inheritance": "Herencia de método",
    "override": "Override de método",
    "dynamic": "Llamada dinámica (reflection)"
}

DEPENDENCY_TYPES = {
    "import": "Importación de módulo",
    "require": "Require de módulo",
    "inheritance": "Herencia de clase",
    "implementation": "Implementación de interfaz",
    "composition": "Composición (tiene-a)",
    "association": "Asociación (usa-a)"
}
```

### 13.3 Formato de Nodos para NodeGraphQt

```python
# Formato interno para NodeGraphQt
@dataclass
class GraphNode:
    id: str                    # unique identifier
    name: str                  # display name
    node_type: str             # class, function, file, group
    file_path: str             # source file
    line_number: int           # line in file
    methods: List[str] = None  # for class nodes
    parameters: List[str] = None  # for function nodes
    complexity: int = 0
    connections: int = 0
    color: str = "#2196F3"     # default color

@dataclass
class GraphConnection:
    source_id: str
    target_id: str
    connection_type: str       # call, dependency, inheritance
    line: int = 0              # line of call/dependency
```

---

## 14. Consideraciones de Seguridad

### 14.1 Principios de Seguridad

```
┌─────────────────────────────────────────────────────────────────────┐
│                 PRINCIPIOS DE SEGURIDAD                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. PRIVACIDAD POR DISEÑO                                          │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Código nunca sale de la máquina local                 │     │
│     │ • No hay telemetry de código fuente                    │     │
│     │ • No hay analytics de qué proyectos analiza el usuario │     │
│     │ • Offline por defecto                                  │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  2. MÍNIMO PRIVILEGIO                                             │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Solo lee archivos del folder seleccionado            │     │
│     │ • No modifica ningún archivo                           │     │
│     │ • No accede a ubicaciones fuera del scope              │     │
│     │ • Sandboxing para análisis                             │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  3. TRANSPARENCIA                                                 │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Código abierto (open source)                         │     │
│     │ • No hay funcionalidades ocultas                       │     │
│     │ • Usuario ve exactamente qué analiza                   │     │
│     │ • Posibilidad de auditar código                        │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 14.2 Matriz de Amenazas

```
┌────────────────────┬────────────────────┬───────────────────────────┐
│     AMENAZA        │    PROBABILIDAD    │        MITIGACIÓN          │
├────────────────────┼────────────────────┼───────────────────────────┤
│ Acceso a archivos  │ Baja (solo folder  │ • Scope limitado a folder  │
│ fuera del proyecto │   seleccionado)    │ • Sandboxing              │
├────────────────────┼────────────────────┼───────────────────────────┤
│ Exfiltración de    │ Muy baja (no hay   │ • Código open source      │
│ código             │   red calls)       │ • Sin APIs externas por   │
│                    │                    │   defecto                 │
├────────────────────┼────────────────────┼───────────────────────────┤
│ Inyección de código│ Baja (solo lectura)│ • Análisis en modo solo   │
│ malicioso en el    │                    │   lectura                 │
│ código analizado   │                    │ • Parser en sandbox       │
├────────────────────┼────────────────────┼───────────────────────────┤
│ Denial of Service  │ Media (archivos    │ • Timeout en análisis     │
│ con archivos       │   muy grandes)     │ • Límite de archivos      │
│ masivos            │                    │ • Memoria limitada        │
├────────────────────┼────────────────────┼───────────────────────────┤
│ Acceso a credenciales│ Muy baja (no hay  │ • Sin almacenamiento de   │
│ en código fuente   │   conexión a BDD)  │   credenciales            │
└────────────────────┴────────────────────┴───────────────────────────┘
```

### 14.3 Configuración de Seguridad

```python
# config.py
class SecurityConfig:
    # Scope de archivos
    MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB max por archivo
    MAX_FILES_PER_ANALYSIS = 1000     # Max 1000 archivos
    ALLOWED_EXTENSIONS = {".py", ".js", ".ts", ".java", ".go", ".rs"}
    
    # Timeout
    ANALYSIS_TIMEOUT = 300  # 5 minutos max
    
    # Memoria
    MAX_MEMORY_MB = 512  # Max 512MB de RAM
    
    # Red (solo si está habilitado)
    ALLOW_NETWORK = False  # Por defecto, sin red
    ALLOWED_EXTERNAL_HOSTS = []  # Vacío por defecto
    
    # Sandbox
    ENABLE_SANDBOX = True  # Sandbox para parsing
```

---

## 15. Rendimiento y Optimización

### 15.1 Benchmarks Estimados

```
┌─────────────────────────────────────────────────────────────────────┐
│                    BENCHMARKS DE RENDIMIENTO                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  PROYECTO PEQUEÑO (< 100 archivos)                                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Lenguaje    │ Archivos │ Tiempo     │ Memoria  │ JSON Size │    │
│  ├─────────────┼──────────┼────────────┼──────────┼───────────┤    │
│  │ Python      │ 50       │ 5-10 seg   │ 100 MB   │ 200 KB    │    │
│  │ JavaScript  │ 80       │ 3-8 seg    │ 80 MB    │ 150 KB    │    │
│  │ TypeScript  │ 60       │ 8-15 seg   │ 120 MB   │ 250 KB    │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  PROYECTO MEDIANO (100-500 archivos)                                │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Lenguaje    │ Archivos │ Tiempo     │ Memoria  │ JSON Size │    │
│  ├─────────────┼──────────┼────────────┼──────────┼───────────┤    │
│  │ Python      │ 200      │ 15-30 seg  │ 200 MB   │ 800 KB    │    │
│  │ JavaScript  │ 300      │ 10-20 seg  │ 150 MB   │ 600 KB    │    │
│  │ TypeScript  │ 250      │ 20-40 seg  │ 250 MB   │ 1 MB      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  PROYECTO GRANDE (500-1000 archivos)                                │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Lenguaje    │ Archivos │ Tiempo     │ Memoria  │ JSON Size │    │
│  ├─────────────┼──────────┼────────────┼──────────┼───────────┤    │
│  │ Python      │ 500      │ 40-60 seg  │ 350 MB   │ 2 MB      │    │
│  │ JavaScript  │ 800      │ 25-45 seg  │ 300 MB   │ 1.5 MB    │    │
│  │ TypeScript  │ 600      │ 50-80 seg  │ 400 MB   │ 2.5 MB    │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 15.2 Estrategias de Optimización

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ESTRATEGIAS DE OPTIMIZACIÓN                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. PARSING PARALELO                                               │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Worker threads con QThreadPool                       │     │
│     │ • QFuture/ QtConcurrent para resultados asíncronos      │     │
│     │ • Señales Qt para comunicar resultados a UI             │     │
│     │ • Speedup: ~Nx para archivos independientes            │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  2. CACHÉ DE RESULTADOS                                            │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Hash de archivo (mtime + size)                       │     │
│     │ • Si no cambió, no re-analizar                        │     │
│     │ • Análisis incremental (re-análisis parcial)           │     │
│     │ • Guardado en: ~/.codemap/cache/                       │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  3. ACTUALIZACIONES REACTIVAS                                      │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Qt Signals/Slots para comunicación hilo-UI            │     │
│     │ • No esperar a terminar todo el análisis               │     │
│     │ • Enviar nodos conforme se generan                     │     │
│     │ • UI actualiza progresivamente                         │     │
│     │ • QProgressDialog para progreso                        │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
│  4. OPTIMIZACIÓN DE MEMORIA                                        │
│     ┌─────────────────────────────────────────────────────────┐     │
│     │ • Releer archivos después de procesar                  │     │
│     │ • Liberar memoria de AST después de extraer datos      │     │
│     │ • Usar generadores para streams grandes                 │     │
│     │ • Limitar profundidad de recursión                     │     │
│     │ • Eliminar referencias cíclicas                        │     │
│     └─────────────────────────────────────────────────────────┘     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 15.3 Configuración de Rendimiento

```python
# performance.py
class PerformanceConfig:
    # Parallelismo
    MAX_WORKERS = None  # None = auto (num_cores)
    CHUNK_SIZE = 50  # Archivos por chunk
    
    # Caché
    ENABLE_CACHE = True
    CACHE_DIR = "~/.codemap/cache"
    CACHE_TTL_DAYS = 7
    
    # Memoria
    MAX_MEMORY_MB = 512
    RELEASE_MEMORY_INTERVAL = 100  # archivos
    
    # Timeout
    FILE_TIMEOUT_SECONDS = 10
    TOTAL_TIMEOUT_SECONDS = 300
    
    # Streaming
    BATCH_SIZE = 20  # Nodos por batch
    PROGRESS_INTERVAL = 0.5  # segundos
```

---

## 16. Casos de Uso Específicos por Tipo de Proyecto

### 16.1 Proyecto Legacy (Monolito)

```
┌─────────────────────────────────────────────────────────────────────┐
│               CASO DE USO: PROYECTO LEGACY                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ESCENARIO:                                                         │
│  Proyecto Python/Django con 5 años, 200 archivos,                   │
│  sin documentación, deuda técnica acumulada.                        │
│                                                                     │
│  PASOS:                                                             │
│  1. Usuario abre folder /var/www/myapp                             │
│  2. CodeMap detecta Python, escanea estructura                     │
│  3. Genera grafo con 200 nodos                                     │
│  4. Usuario filtra por "most connected"                            │
│  5. Encuentra clases con alta conectividad                         │
│  6. Ve call graph de funciones críticas                            │
│  7. Identifica tight coupling                                      │
│  8. Guarda insights para refactorización                           │
│                                                                     │
│  INSIGHTS ESPERADOS:                                                │
│  • "La clase OrderProcessor tiene 47 conexiones"                   │
│  • "validate_payment se llama desde 23 lugares"                    │
│  • "models.py importa 15 módulos - considerar拆分"                  │
│  • "Hay 3 funciones con complejidad > 50"                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 16.2 Microservicios

```
┌─────────────────────────────────────────────────────────────────────┐
│              CASO DE USO: MICROSERVICIOS                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ESCENARIO:                                                         │
│  Repo con 5 microservicios (Python, Node, Go),                     │
│  communication complex entre servicios.                             │
│                                                                     │
│  PASOS:                                                             │
│  1. Usuario abre folder /repo/microservices                        │
│  2. CodeMap detecta múltiples lenguajes                            │
│  3. Analiza cada servicio por separado                             │
│  4. Identifica llamadas entre servicios (API clients)              │
│  5. Muestra grafo de dependencias entre servicios                  │
│  6. Identifica acoplamiento circular                               │
│  7. Detecta duplicated code entre servicios                        │
│                                                                     │
│  INSIGHTS ESPERADOS:                                                │
│  • "user-service ↔ payment-service: acoplamiento circular"         │
│  • "5 funciones duplicadas entre auth y user service"              │
│  • "notification-service no debería importar de order-service"     │
│  • "API client en order-service llama 30 endpoints de payment"     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 16.3 Proyecto Nuevo (Onboarding)

```
┌─────────────────────────────────────────────────────────────────────┐
│            CASO DE USO: PROYECTO NUEVO (ONBOARDING)                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ESCENARIO:                                                         │
│  Desarrollador nuevo en proyecto con 3 meses de edad,               │
│  necesita entender estructura rápidamente.                          │
│                                                                     │
│  PASOS:                                                             │
│  1. Abre proyecto en CodeMap                                       │
│  2. Ve vista de árbol de directorios                               │
│  3. Explora arquitectura por carpetas                              │
│  4. Busca "Controller" o "View" para entender flujo                │
│  5. Click en clase central para ver métodos                        │
│  6. Sigue el call flow de una request típica                        │
│  7. Anota dependencias importantes                                 │
│                                                                     │
│  INSIGHTS ESPERADOS:                                                │
│  • "Estructura clara: controllers/ → services/ → models/"          │
│  • "Entry point: __main__.py → QApplication()"                        │
│  • "Request típico: router.post('/users') → UserService.create()"  │
│  • "Database: usa SQLAlchemy, conexión en db.py"                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 17. Instalación y Despliegue

### 17.1 Requisitos del Sistema

```
┌─────────────────────────────────────────────────────────────────────┐
│                    REQUISITOS DEL SISTEMA                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  MÍNIMOS:                                                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • OS: Linux, macOS, Windows                                 │    │
│  │ • CPU: 2 cores                                              │    │
│  │ • RAM: 2 GB                                                 │    │
│  │ • Disco: 500 MB libres                                      │    │
│  │ • Python: 3.11+                                             │    │
│  │ • uv: última versión (https://uv.ei.ai/)                   │    │
│  │ • Qt6/PyQt6 dependencies del sistema                        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  RECOMENDADOS:                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • OS: Linux o macOS                                         │    │
│  │ • CPU: 4+ cores                                             │    │
│  │ • RAM: 4 GB+                                                │    │
│  │ • Disco: 1 GB libres                                        │    │
│  │ • Python: 3.12+                                             │    │
│  │ • uv: última versión                                        │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  PARA PROYECTOS GRANDES (>500 archivos):                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • CPU: 8+ cores                                             │    │
│  │ • RAM: 8 GB+                                                │    │
│  │ • SSD recomendado                                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 17.2 Instalación con uv (Método Recomendado)

```bash
# 1. Instalar uv (si no lo tienes)
curl -LsSf https://uv.ei.ai/installer | sh

# 2. Clonar o descargar CodeMap
git clone https://github.com/codemap-dev/codemap.git
cd codemap

# 3. Ejecutar directamente (sin instalación)
uv run codemap ui

# 4. O instalar como comando global
uv tool install .

# 5. Verificar instalación
codemap --version

# 6. Lanzar aplicación desktop
codemap ui

# 7. O ejecutar análisis desde CLI
codemap analyze /path/to/project
```

### 17.3 Instalación en Entorno Virtual

```bash
# 1. Clonar repositorio
git clone https://github.com/codemap-dev/codemap.git
cd codemap

# 2. Crear entorno virtual con uv
uv venv
source .venv/bin/activate  # Linux/macOS
# o
.venv\Scripts\activate     # Windows

# 3. Instalar dependencias
uv sync

# 4. Ejecutar aplicación desktop
python -m codemap ui

# 5. O instalar como editable
pip install -e .
codemap --help
```

### 17.4 Build de Ejecutable (PyInstaller)

```bash
# Instalar dependencias de build
uv sync --extra build

# Generar ejecutable
python scripts/build.sh

# El ejecutable estará en: dist/codemap
./dist/codemap ui

# Para macOS (.app)
python scripts/build.sh --macos-app

# Para Windows (.exe)
python scripts/build.sh --windows
```

```bash
# scripts/build.sh
#!/bin/bash
set -e

# Instalar PyInstaller
pip install pyinstaller

# Build para plataforma actual
pyinstaller --onefile \
    --name codemap \
    --collect-all NodeGraphQt \
    --collect-all PyQt6 \
    --collect-all tree_sitter \
    --hidden-import NodeGraphQt \
    --hidden-import PyQt6 \
    codemap/__main__.py

echo "Ejecutable generado en: dist/codemap"
```

### 17.5 Uso desde CLI

```bash
# Ver ayuda
codemap --help

# Lanzar aplicación desktop
codemap ui

# Analizar un proyecto
codemap analyze /path/to/project

# Analizar y guardar JSON
codemap analyze /path/to/project --output result.json

# Analizar solo Python
codemap analyze /path/to/project --language python

# Con métricas de IA
codemap analyze /path/to/project --with-ai

# Modo verbose
codemap analyze /path/to/project --verbose

# Verificar salud del sistema
codemap doctor
```

### 17.6 Inicio Rápido

```
┌─────────────────────────────────────────────────────────────────────┐
│                   GUÍA DE INICIO RÁPIDO                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Paso 1: Instalar uv (si no tienes)                                │
│  ─────────────────────────────────────────────────────────────     │
│  $ curl -LsSf https://uv.ei.ai/installer | sh                       │
│                                                                     │
│  Paso 2: Lanzar aplicación desktop                                 │
│  ─────────────────────────────────────────────────────────────     │
│  $ uv run codemap ui                                                │
│                                                                     │
│  Paso 3: Seleccionar folder a analizar                             │
│  ─────────────────────────────────────────────────────────────     │
│  File → Open Folder → Elige tu proyecto                            │
│  CodeMap analizará automáticamente                                 │
│                                                                     │
│  ¡Listo! Tu código visualizado en editor de nodos                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

# Modo verbose para debugging
codemap analyze /path/to/project --verbose

# Ver ayuda
codemap --help
```

---

## 18. Guía de Contribución

### 18.1 Configuración de Desarrollo (100% Python)

```bash
# 1. Fork y clonar
git clone https://github.com/TUUSUARIO/codemap.git
cd codemap

# 2. Instalar uv (si no lo tienes)
curl -LsSf https://uv.ei.ai/installer | sh

# 3. Crear rama para feature
git checkout -b feature/mi-nueva-feature

# 4. Instalar en modo desarrollo
uv sync --extra dev

# 5. Instalar pre-commit hooks
uv run pre-commit install

# 6. Verificar instalación
uv run codemap --help

# 7. Ejecutar tests
uv run pytest

# 8. Ejecutar linters
uv run ruff check .
uv run mypy .
uv run black --check .

# 9. Crear commit
git add .
git commit -m "feat: descripción de mi feature"

# 10. Push y crear PR
git push origin feature/mi-nueva-feature
```

### 18.2 Estilo de Código (Python Only)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    GUÍA DE ESTILO (PYTHON)                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  PYTHON:                                                            │
│  • Seguir PEP 8                                                     │
│  • Usar ruff como linter                                            │
│  • Type hints obligatorios para funciones públicas                  │
│  • Docstrings en formato Google                                    │
│  • Longitud máxima: 100 caracteres                                  │
│  • Imports: usar uv para organizar                                  │
│                                                                     │
│  PYQT6 + NODEGRAPHQT:                                                │
│  • Usar NodeGraphQt para grafos y nodos                             │
│  • Separar lógica de UI en servicios                                │
│  • Signals/slots de Qt para eventos                                 │
│  • Custom widgets heredando de NodeGraphQt                          │
│                                                                     │
│  COMMITS:                                                           │
│  • Conventional Commits: type(scope): descripción                   │
│  • Types: feat, fix, docs, style, refactor, test, chore            │
│  • Max 72 caracteres en subject                                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 18.3 Áreas de Contribución

```
┌─────────────────────────────────────────────────────────────────────┐
│                 ÁREAS DONDE CONTRIBUIR                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ALTA PRIORIDAD:                                                    │
│  • Nuevos language parsers (Go, Rust, C++, Ruby, PHP)              │
│  • Mejoras en call graph (precisión, velocidad)                    │
│  • Optimizaciones de rendimiento                                   │
│  • Mejoras de UI (NodeGraphQt)                                      │
│  • Mejores insights de IA                                          │
│                                                                     │
│  MEDIA PRIORIDAD:                                                   │
│  • Integraciones CLI                                                │
│  • Export formats (SVG, PNG, JSON)                                 │
│  • Mejores métricas de calidad de código                           │
│  • Detección de patrones y antipatrones                            │
│  • Tests adicionales                                               │
│                                                                     │
│  BAJA PRIORIDAD:                                                    │
│  • Internacionalización (i18n)                                     │
│  • Temas (dark mode, colores personalizables)                      │
│  • Documentación (tutorials, examples)                             │
│  • PyInstaller builds para diferentes plataformas                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 18.4 Testing

```bash
# Ejecutar todos los tests
uv run pytest

# Tests con coverage
uv run pytest --cov=codemap --cov-report=html

# Tests específicos
uv run pytest tests/test_analyzers/
uv run pytest tests/test_services/
uv run pytest tests/test_ui/

# Tests en modo watch
uv run pytest --watch
```

---

## 19. Troubleshooting

### 19.1 Problemas Comunes

```
┌─────────────────────────────────────────────────────────────────────┐
│                 PROBLEMAS Y SOLUCIONES                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  PROBLEMA: "uv: command not found"                                 │
│  CAUSA: uv no está instalado                                       │
│  SOLUCIÓN:                                                          │
│  $ curl -LsSf https://uv.ei.ai/installer | sh                       │
│  $ export PATH="$HOME/.uv/bin:$PATH"                               │
│                                                                     │
│  PROBLEMA: "Out of Memory" en proyectos grandes                     │
│  CAUSA: El proyecto tiene muchos archivos (>1000)                  │
│  SOLUCIÓN:                                                          │
│  • Usar flag --max-files para limitar                              │
│  • Aumentar memoria en config                                      │
│  • Analizar por carpetas individuales                              │
│                                                                     │
│  PROBLEMA: Análisis muy lento                                      │
│  CAUSA: HDD en lugar de SSD, o muchos archivos                     │
│  SOLUCIÓN:                                                          │
│  • Mover proyecto a SSD                                            │
│  • Usar análisis incremental si hay caché                          │
│  • Reducir languages si no son necesarios                          │
│  • Verificar que uv está en PATH                                   │
│                                                                     │
│  PROBLEMA: No detecta lenguaje                                     │
│  CAUSA: Archivos sin extensión o con extensión rara               │
│  SOLUCIÓN:                                                          │
│  • Usar flag --language-force python                               │
│  • Verificar que los archivos tienen extensión                     │
│                                                                     │
│  PROBLEMA: Call graph incompleto                                   │
│  CAUSA: dynamic imports, reflection, decorators complejos          │
│  SOLUCIÓN:                                                          │
│  • Los dynamic calls no pueden detectarse automáticamente          │
│  • Usar anotaciones manuales si es crítico                         │
│  • Reportar issue si es un patrón común                            │
│                                                                     │
│  PROBLEMA: Frontend no carga                                       │
│  CAUSA: Puerto ocupado o build incompleto                          │
│  SOLUCIÓN:                                                          │
│  • Verificar que el puerto 8000 está libre                         │
│  • Reinstalar: uv sync                                             │
│  • Ver logs: codemap serve --verbose                               │
│  • Verificar salud: codemap doctor                                 │
│                                                                     │
│  PROBLEMA: PyInstaller build falla                                 │
│  CAUSA: Dependencias faltantes                                     │
│  SOLUCIÓN:                                                          │
│  • Verificar que pyproject.toml tiene todos los imports            │
│  • Instalar deps de build: uv sync --extra build                   │
│  • Ver logs de PyInstaller                                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 19.2 Logs y Debugging

```bash
# Habilitar logs verbose
codemap analyze /path --verbose --log-level debug

# Ver logs del servidor
codemap serve --verbose --log-level debug

# Generar report de debugging
codemap doctor

# Ver información del sistema
codemap doctor --verbose

# Verificar instalación de uv
uv --version

# Verificar entorno Python
python --version
uv run python --version
```

### 19.3 Diagnóstico de Salud del Sistema

```

*Documento de Arquitectura Técnica - CodeMap v3.0*
*Última actualización: Diciembre 2025*
*Versión del documento: 3.1 - AI Providers Edition*

```
┌─────────────────────────────────────────────────────────────────────┐
│                     CODEMAP - RESUMEN                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Herramienta de análisis y visualización de código                  │
│  100% Python + uv + NodeGraphQt + PyQt6                             │
│  Con soporte para IA configurable (Ollama, OpenAI, Anthropic)      │
│                                                                     │
│  INSTALACIÓN:                                                       │
│  $ curl -LsSf https://uv.ei.ai/installer | sh                       │
│  $ uv run codemap ui                                                │
│                                                                     │
│  USO:                                                               │
│  $ codemap ui                 # Aplicación desktop                  │
│  $ codemap analyze /path      # CLI análisis                       │
│                                                                     │
│  IA PROVIDERS:                                                      │
│  • Ollama (local, gratis, offline) - RECOMENDADO                   │
│  • OpenAI (cloud, mejor calidad)                                    │
│  • Anthropic (cloud, razonamiento complejo)                         │
│  • APIs compatibles OpenAI (Groq, Together, etc.)                   │
│  • Fallback local (análisis estático sin IA)                        │
│                                                                     │
│  CARACTERÍSTICAS:                                                   │
│  ✓ Sin configuración                                                │
│  ✓ Sin copiar código fuente                                         │
│  ✓ Privacidad total (100% local, offline)                           │
│  ✓ Multi-lenguaje (Python, JS, TS, Java...)                        │
│  ✓ Call graph interactivo                                           │
│  ✓ IA configurable por el usuario                                   │
│  ✓ 100% Python (NodeGraphQt + PyQt6 Desktop App)                    │
│  ✓ Editor de nodos nativo (NodeGraphQt)                             │
│  ✓ Sin navegador, sin servidor, sin internet (con Ollama)          │
│  ✓ Instalación simple con uv                                        │
│                                                                     │
│  REPOSITORIO: https://github.com/codemap-dev/codemap                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 20. Roadmap Visual

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TIMELINE DE DESARROLLO                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  v1.0 - MVP                         ████████████████████  100%      │
│  ┌────────────────────────────┐     Terminada                       │
│  │ • Python, JS, TS, Java     │                                    │
│  │ • Call graph básico        │                                    │
│  │ • Grafo NodeGraphQt          │                                    │
│  │ • CLI + API local          │                                    │
│  │ • 100% local               │                                    │
│  └────────────────────────────┘                                    │
│                                                                     │
│  v1.1 - Estabilidad              ████████████████░░░░░  80%         │
│  ┌────────────────────────────┐     En progreso                     │
│  │ • Bug fixes                │                                    │
│  │ • Optimizaciones           │                                    │
│  │ • Mejoras de UI            │                                    │
│  │ • Tests adicionales        │                                    │
│  └────────────────────────────┘                                    │
│                                                                     │
│  v2.0 - Multi-lenguaje           ██████████░░░░░░░░░░░  30%         │
│  ┌────────────────────────────┐     Planificado Q2 2026            │
│  │ • Go, Rust, C++, Ruby      │                                    │
│  │ • Análisis incremental     │                                    │
│  │ • File watching            │                                    │
│  │ • Export SVG/PNG           │                                    │
│  └────────────────────────────┘                                    │
│                                                                     │
│  v3.0 - Colaboración             ████░░░░░░░░░░░░░░░░░  10%         │
│  ┌────────────────────────────┐     Planificado Q4 2026            │
│  │ • GitHub/GitLab integration│                                    │
│  │ • Análisis de diffs        │                                    │
│  │ • Teams support            │                                    │
│  │ • VS Code plugin           │                                    │
│  │ • API pública              │                                    │
│  └────────────────────────────┘                                    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 21. Licencia y Contacto

### 21.1 Licencia

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LICENCIA                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  CodeMap está licenciado bajo MIT License:                         │
│                                                                     │
│  MIT License Copyright (c) 2025 CodeMap                            │
│                                                                     │
│  PERMITE:                                                          │
│  ✓ Uso comercial y personal                                        │
│  ✓ Modificación del código                                         │
│  ✓ Distribución                                                    │
│  ✓ Uso privado                                                     │
│                                                                     │
│  REQUIERE:                                                         │
│  • Incluir copyright notice                                        │
│  • Incluir license notice                                          │
│                                                                     │
│  PROHÍBE:                                                          │
│  • Uso del nombre del proyecto para endorsement                    │
│                                                                     │
│  El código de análisis es 100% local, no hay restricciones         │
│  adicionales por analizar código propietario.                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 21.2 Contacto y Recursos

```
┌─────────────────────────────────────────────────────────────────────┐
│                    RECURSOS Y CONTACTO                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  REPOSITORIO:                                                       │
│  • GitHub: https://github.com/codemap/codemap                      │
│  • Issues: https://github.com/codemap/codemap/issues               │
│  • Discussions: https://github.com/codemap/codemap/discussions     │
│                                                                     │
│  DOCUMENTACIÓN:                                                     │
│  • Docs: https://codemap.dev/docs                                  │
│  • API: https://codemap.dev/api                                    │
│  • Examples: https://codemap.dev/examples                          │
│                                                                     │
│  COMUNIDAD:                                                         │
│  • Discord: https://discord.gg/codemap                             │
│  • Twitter: @codemap_dev                                           │
│  • Email: hello@codemap.dev                                        │
│                                                                     │
│  CONTRIBUCIÓN:                                                      │
│  • Contributing guide: CONTRIBUTING.md                              │
│  • Code of conduct: CODE_OF_CONDUCT.md                             │
│  • Security: SECURITY.md                                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 22. Anexo: Ejemplo de Análisis Completo

### 22.1 Proyecto de Ejemplo

```
~/my-flask-app/
├── app.py                    (52 líneas)
├── config.py                 (45 líneas)
├── requirements.txt
├── models/
│   ├── __init__.py
│   ├── user.py               (120 líneas)
│   └── order.py              (200 líneas)
├── routes/
│   ├── __init__.py
│   ├── auth.py               (180 líneas)
│   └── api.py                (250 líneas)
├── services/
│   ├── __init__.py
│   ├── email.py              (80 líneas)
│   └── payment.py            (150 líneas)
└── utils/
    ├── __init__.py
    └── helpers.py            (95 líneas)

Total: 8 archivos Python, ~1172 líneas de código
```

### 22.2 Resultado del Análisis

```json
{
  "summary": {
    "files": 8,
    "loc": 1172,
    "classes": 12,
    "functions": 45,
    "complexity_avg": 5.2,
    "analysis_time_seconds": 12.5
  },
  "structure": {
    "entities": [
      {"id": "cls:User", "file": "models/user.py", "line": 10, "methods": 8},
      {"id": "cls:Order", "file": "models/order.py", "line": 15, "methods": 12},
      {"id": "func:login", "file": "routes/auth.py", "line": 25, "complexity": 8},
      {"id": "func:process_payment", "file": "services/payment.py", "line": 10, "complexity": 15}
    ]
  },
  "call_graph": {
    "hotspots": [
      {"function": "process_payment", "callers": 5, "callees": 8, "complexity": 15},
      {"function": "login", "callers": 3, "callees": 6, "complexity": 8}
    ]
  },
  "dependencies": {
    "most_connected": [
      {"entity": "User", "imports": 5, "imported_by": 7},
      {"entity": "Order", "imports": 4, "imported_by": 6}
    ]
  },
  "insights": [
    "La función process_payment tiene complejidad alta (15)",
    "User es el modelo más conectado - considerar reducir acoplamiento",
    "Order depende de User - posible violation de DIP",
    "Hay 3 funciones con más de 10 params - considerar grouping"
  ]
}
```

### 22.3 Visualización Generada

```
┌─────────────────────────────────────────────────────────────────────┐
│                    GRAFO DE VISUALIZACIÓN                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│                    ┌─────────────┐                                  │
│                    │   app.py    │                                  │
│                    │   (entry)   │                                  │
│                    └──────┬──────┘                                  │
│                           │                                         │
│         ┌─────────────────┼─────────────────┐                      │
│         │                 │                 │                      │
│         ▼                 ▼                 ▼                      │
│   ┌───────────┐    ┌───────────┐    ┌───────────┐                 │
│   │ routes/   │    │ models/   │    │ services/ │                 │
│   │ auth.py   │───▶│ user.py   │◀───│ payment.py│                 │
│   └─────┬─────┘    └─────┬─────┘    └─────┬─────┘                 │
│         │                │                │                         │
│         │         ┌──────┴──────┐         │                         │
│         │         │             │         │                         │
│         ▼         ▼             ▼         ▼                         │
│   ┌───────────┐ ┌───────────┐ ┌───────────┐                        │
│   │ routes/   │ │ routes/   │ │ services/ │                        │
│   │ api.py    │ │ auth.py   │ │ email.py  │                        │
│   └───────────┘ └───────────┘ └───────────┘                        │
│                                                                     │
│   Leyenda:                                                          │
│   ● Nodo = Clase/Función                                           │
│   ──────► = Dependencia/Llamada                                    │
│   Tamaño = Complejidad                                             │
│   Color  = Tipo (azul=model, verde=route, naranja=service)         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

*Documento de Arquitectura Técnica - CodeMap v2.0*
*Última actualización: Diciembre 2025*
*Versión del documento: 3.1 - AI Providers Edition*

```
┌─────────────────────────────────────────────────────────────────────┐
│                     CODEMAP - RESUMEN                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Herramienta de análisis y visualización de código                  │
│  100% Python + uv + NodeGraphQt + PyQt6 Desktop App                │
│  Con soporte para IA configurable (Ollama, OpenAI, Anthropic)      │
│                                                                     │
│  INSTALACIÓN:                                                       │
│  $ curl -LsSf https://uv.ei.ai/installer | sh                       │
│  $ uv run codemap ui                                                │
│                                                                     │
│  USO:                                                               │
│  $ codemap ui                 # Aplicación desktop                  │
│  $ codemap analyze /path      # CLI análisis                       │
│                                                                     │
│  IA PROVIDERS:                                                      │
│  • Ollama (local, gratis, offline) - RECOMENDADO                   │
│  • OpenAI (cloud, mejor calidad)                                    │
│  • Anthropic (cloud, razonamiento complejo)                         │
│  • APIs compatibles OpenAI (Groq, Together, etc.)                   │
│  • Fallback local (análisis estático sin IA)                        │
│                                                                     │
│  CARACTERÍSTICAS:                                                   │
│  ✓ Sin configuración                                                │
│  ✓ Sin copiar código fuente                                         │
│  ✓ Privacidad total (100% local, offline)                           │
│  ✓ Multi-lenguaje (Python, JS, TS, Java...)                        │
│  ✓ Call graph interactivo                                           │
│  ✓ IA configurable por el usuario                                   │
│  ✓ 100% Python (NodeGraphQt + PyQt6 Desktop App)                    │
│  ✓ Editor de nodos nativo (NodeGraphQt)                             │
│  ✓ Sin navegador, sin servidor, sin internet (con Ollama)          │
│  ✓ Instalación simple con uv                                        │
│                                                                     │
│  REPOSITORIO: https://github.com/codemap-dev/codemap                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```
