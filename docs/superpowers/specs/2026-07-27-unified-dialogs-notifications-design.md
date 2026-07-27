# Unified Dialogs and Notifications Design

## Purpose

HAYA-TAB currently gives ten dialogs a common visual surface through the global
`.modal` class, but individual components override width and height rules. The
result is content-dependent desktop sizing, mobile overflow, and notification
surfaces with unrelated spacing and decoration.

This change introduces durable UI primitives so new dialogs and notifications
inherit a consistent, responsive design without component-specific sizing CSS.

## Scope

The change covers:

- All ten dialogs mounted by `App.vue`.
- The ordinary toast and the persistent sync-task toast.
- Desktop and narrow mobile viewport behavior.
- Dialog semantics, Escape handling, overlay-click handling, focus entry, and
  focus restoration.
- Automated browser regression coverage for shared styling and containment.

Context menus, batch action bars, viewer toolbars, and MIDI-learning overlays
remain out of scope because their interaction patterns are not dialogs or
notifications.

## Approaches Considered

### CSS-only patch

Centralizing widths in `style.css` would fix the current screenshots quickly,
but every dialog would still duplicate overlay and surface markup. It would not
prevent future scoped styles from overriding the shared rules.

### Shared modal component and notification tokens

This is the selected approach. A shared component owns dialog structure and
behavior, semantic size classes provide predictable dimensions, and common CSS
custom properties align notification styling while preserving each
notification's distinct lifecycle.

### General popup framework

A single abstraction for dialogs, notifications, menus, and floating toolbars
would be broader than the defect and would conflate different keyboard and
positioning behavior. It is intentionally rejected.

## Modal Architecture

Create `frontend/src/components/common/BaseModal.vue`.

The component accepts:

- `open`: whether the dialog is rendered.
- `title`: visible dialog heading.
- `size`: `small`, `medium`, or `large`.
- `contentClass`: optional class applied to the content region.
- `closeOnOverlay`: defaults to `true`.

It emits `close` and exposes:

- A default slot for dialog content.
- An `actions` slot for the footer.

The component owns:

- The fixed overlay.
- The dialog surface.
- Heading markup and generated heading ID.
- `role="dialog"` and `aria-modal="true"`.
- `aria-labelledby` association.
- Escape-to-close behavior.
- Overlay-click-to-close behavior.
- Initial focus on the dialog surface when opened.
- Restoration of the previously focused element when closed.

The component does not own business actions, loading state, or data. Existing
modal components retain those responsibilities and call their current store
hide actions when `BaseModal` emits `close`.

## Modal Sizing

Modal surfaces use `box-sizing: border-box`, `min-width: 0`, and a viewport-safe
width:

```css
width: min(var(--modal-width), calc(100vw - 32px));
max-height: calc(100dvh - 32px);
```

The semantic widths are:

- Small: 420px.
- Medium: 500px.
- Large: 800px.

Assignments:

- Small: Category, Move, Batch Move, Confirm, Plugin Settings.
- Medium: Edit Tab, Key Bindings, Cloud Upload, WebDAV.
- Large: Cloud Picker.

The large Cloud Picker keeps a tall working area, but its height is bounded by
the same viewport margin. Modal content scrolls inside the surface while the
heading and action footer remain visible.

Every viewport must retain at least 16px horizontal and vertical space between
the dialog and the viewport edge. No dialog may use a component-scoped width,
minimum width, maximum width, or outer height.

## Shared Dialog Layout

The shared layout uses three regions:

- Header: consistent heading typography and spacing.
- Body: flexible, scrollable content with a consistent gap from the header.
- Footer: right-aligned actions with a consistent gap and top spacing.

Dialogs that need a leading action, such as WebDAV connection testing, use an
explicit `.modal-actions__leading` wrapper rather than inline margin styles.
Confirm-dialog alternate actions use the same wrapper.

Forms keep their existing submit behavior. Submit buttons in the shared footer
use the HTML `form` attribute when the form itself lives in the body slot.

## Notification Design

Ordinary and sync-task notifications keep their different behavior:

- Ordinary notifications remain transient and stackable.
- Sync-task notifications remain persistent, collapsible, and status-aware.

Both consume shared custom properties defined in `style.css`:

```css
--notification-radius: 8px;
--notification-padding-block: 12px;
--notification-padding-inline: 16px;
--notification-max-width: 320px;
--notification-border-width: 1px;
--notification-accent-width: 4px;
--notification-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
```

Both use `box-sizing: border-box` and:

```css
width: fit-content;
max-width: min(var(--notification-max-width), calc(100vw - 48px));
```

The different z-index values remain because sync progress must stay visible
above modal-independent notifications. Color continues to communicate success,
warning, and error status.

## Error and Edge Behavior

- A dialog close request continues to call the existing store action, so
  unsaved form state is cleared exactly as it is today.
- Buttons disabled by loading state remain unchanged.
- Large content scrolls without moving the footer off-screen.
- At 375x667 and smaller supported viewports, every dialog remains fully inside
  the viewport with the required margin.
- Long notification text wraps inside the responsive maximum width.
- The modal restores focus only when the previously focused element remains
  connected to the document.

## Testing

Add a Playwright modal regression suite that uses the existing Pinia UI store to
open each real dialog without file-picker or network side effects.

Tests verify:

- All ten dialogs use the shared dialog surface.
- Each dialog has the intended semantic size.
- Overlay, surface, header, body, and footer styles are shared.
- Desktop widths match their semantic variants.
- All dialogs remain within a 375x667 viewport with at least 16px margins.
- Dialog semantics and heading association are present.
- Escape emits the existing close path.
- Focus enters the dialog and returns to the triggering element.
- Ordinary and sync-task notifications resolve the same shared design tokens.
- Long notification content remains viewport-contained.

Run the targeted regression suite first, then the frontend TypeScript/Vite
build, followed by the existing E2E suite.

## Success Criteria

- No modal component defines its own outer width, minimum width, maximum width,
  padding, radius, overlay, or shadow.
- All dialogs use `BaseModal`.
- Desktop sizing is determined only by `small`, `medium`, and `large`.
- All dialogs fit within the tested mobile viewport with 16px margins.
- Both notification types visibly share spacing, radius, border, shadow, and
  responsive width tokens.
- Existing business behavior and localization remain unchanged.
- New regression tests and the existing frontend build pass.
