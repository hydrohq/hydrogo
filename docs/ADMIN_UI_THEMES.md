# Admin Studio visual modes

HydroGo Studio keeps the visual layer independent from the server-rendered
admin screens. The existing HydroGo and shadcn modes remain available; the
Workbench mode adds a denser, enterprise-oriented presentation for daily
content, commerce, and operations work.

## Available modes

- **HydroGo**: the original branded Studio visual language.
- **shadcn**: neutral zinc surfaces with compact controls and light/dark variants.
- **Workbench**: a restrained blue-gray workspace with a dark navigation rail,
  stronger data hierarchy, quieter surfaces, and accessible focus states.

The theme button cycles through light and dark variants of each mode and stores
the choice in `localStorage` under `hydro-ui-theme`. The mode changes tokens and
shared semantic selectors only; it does not change routes, permissions, or
module behavior.

## Why not ship a full component framework?

Studio is server-rendered Go HTML with a strict CSP and no required frontend
runtime. Radix-style primitives are a useful accessibility reference, while
Carbon, Material, Fluent, and Ant Design are valid design references for future
Client Editions. Importing their full React runtimes into Studio would add
weight and duplicate behavior already provided by the server-rendered shell.
