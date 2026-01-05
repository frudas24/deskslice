# ANEXO — Contexto, Propósito y Comportamiento Esperado

## 1) Contexto / Razón de existir

Este proyecto nace de una limitación práctica:
**la extensión oficial de Codex en VS Code no está diseñada para uso remoto ni móvil**, pero el flujo de trabajo moderno sí lo exige (revisar, ajustar o lanzar acciones desde el celular sin sentarse frente al PC).

En lugar de:

* reimplementar Codex,
* interceptar APIs internas,
* o modificar/extender la extensión oficial (frágil, acoplado),

se decide una solución **agnóstica, robusta y de mínima fricción**:

> **Operar directamente la interfaz real de Codex dentro de VS Code**,
> usando streaming de video + inyección de input, como un *remote desktop especializado*.

La idea no es “copiar” la UI de Codex, sino **usar la original**, exactamente como fue diseñada.

---

## 2) ¿Para qué sirve?

Este sistema permite:

* Usar **Codex tal cual o similares** (sin wrappers, sin clones).
* Operarlo desde un **celular en la LAN**:

  * revisar salidas,
  * escribir prompts,
  * hacer scroll,
  * lanzar acciones.
* Hacerlo con **latencia baja** (WebRTC), usable incluso para interacción continua.
* Evitar dependencias en APIs privadas o cambios futuros del plugin.

En la práctica:

* el PC sigue siendo el “host de verdad”,
* el celular es solo un **control remoto táctil inteligente**.

---

## 3) Qué NO es (decisiones explícitas)

Para evitar confusiones futuras, este sistema **NO**:

* ❌ Persiste chats, prompts o respuestas.
* ❌ Interpreta ni scrapea el contenido de Codex.
* ❌ Reimplementa la lógica del agente.
* ❌ Intenta ser multiusuario o colaborativo.
* ❌ Hace “magia” para detectar inputs o layouts.

Todo lo anterior es deliberado:
menos acoplamiento = más estabilidad a largo plazo.

---

## 4) Filosofía de diseño

### 4.1 Control explícito > Automatización frágil

* El usuario **calibra manualmente**:

  * qué parte del escritorio es el panel de Codex,
  * dónde está el input,
  * dónde se hace scroll.
* Si el layout cambia, **se vuelve a calibrar**.
* No hay heurísticas, OCR ni detección automática.

Esto hace que:

* el sistema sea predecible,
* el comportamiento sea obvio,
* y los fallos sean visibles inmediatamente (a simple vista).

---

### 4.2 Remote desktop, pero *surgical*

No se transmite:

* todo el escritorio,
* ventanas innecesarias,
* ni información fuera del panel de interés.

Se transmite **solo lo necesario**:

* en presetup: fullscreen (solo para calibrar),
* en run: **un recorte preciso** del panel Codex.

Eso reduce:

* distracciones,
* consumo de ancho de banda,
* y errores de input.

---

### 4.3 Seguridad pragmática

El modelo de seguridad es intencionalmente simple:

* **Password único** en `.env`:

  * evita accesos accidentales,
  * protege en LAN compartida.
* No hay usuarios, sesiones complejas ni tokens.
* Si en el futuro se expone a WAN:

  * se hace vía **WireGuard**,
  * sin cambiar una sola línea del código de la app.

Seguridad por **perímetro**, no por complejidad interna.

---

## 5) Cómo debería funcionar “al final” (visión operativa)

### Flujo normal de uso

1. El servicio corre en el PC.
2. Desde el celular:

   * abres `http://PC:8787`,
   * ingresas el password.
3. Entras a **Presetup (Fullscreen)**:

   * eliges el monitor correcto,
   * trazas el rect del panel Codex,
   * trazas el rect del input,
   * trazas el rect scrolleable.
4. Cambias a **Run (Cropped)**:

   * solo ves el panel Codex,
   * el stream es fluido.
5. Interactúas:

   * tocas el input → se enfoca,
   * escribes desde el teclado del celular,
   * haces scroll con drag natural,
   * observas las respuestas en tiempo real.
6. Si algo cambia (VS Code se mueve, layout distinto):

   * presionas **Restart Presetup**,
   * recalibras en segundos,
   * sigues trabajando.

---

## 6) Resultado esperado

Al finalizar este proyecto, el sistema debe sentirse como:

* **“tener Codex en el bolsillo”**,
  sin sacrificar:

  * fidelidad,
  * estabilidad,
  * ni control.

No es una app nueva.
No es un wrapper.
No es un hack frágil.

Es simplemente **una extensión natural del escritorio**, llevada al móvil con criterio técnico y mínimo acoplamiento.

---

Si quieres, el siguiente paso natural es:

* convertir este TODO + anexo en un **README.md inicial**, o
* arrancar directamente por el **Dominio 1–5** (calib + mapper + gestures) con código y tests.
