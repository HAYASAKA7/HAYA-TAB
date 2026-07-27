# Unified Dialogs and Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace duplicated modal shells and drifting popup sizes with one responsive, accessible component and align notification surfaces through shared design tokens.

**Architecture:** `BaseModal.vue` owns dialog structure, behavior, accessibility, focus, and semantic sizing. Feature modals retain business state and render content/actions through slots. A global `notification-surface` class and CSS custom properties align transient and sync-task notification visuals without merging their lifecycles.

**Tech Stack:** Vue 3, TypeScript, Pinia, CSS custom properties, Playwright, Vite.

---

### Task 1: Add failing dialog regression coverage

**Files:**
- Create: `frontend/tests/e2e/modal-consistency.spec.ts`

- [ ] **Step 1: Write the real-dialog browser test**

Create helpers that access the mounted Pinia UI store through Vue's app
instance, close all dialogs, set safe local WebDAV values, and open each of the
ten real dialogs. Define the expected size mapping:

```ts
const dialogCases = [
  ['edit', 'medium'],
  ['category', 'small'],
  ['move', 'small'],
  ['batchMove', 'small'],
  ['confirm', 'small'],
  ['keyBindings', 'medium'],
  ['cloudPicker', 'large'],
  ['cloudUpload', 'medium'],
  ['webdav', 'medium'],
  ['plugin', 'small'],
] as const
```

For every case, assert that `[data-testid="base-modal"]` is visible and its
`data-modal-size` matches the expected value. At `375x667`, assert:

```ts
expect(box.x).toBeGreaterThanOrEqual(15)
expect(box.y).toBeGreaterThanOrEqual(15)
expect(box.x + box.width).toBeLessThanOrEqual(360)
expect(box.y + box.height).toBeLessThanOrEqual(652)
```

Also assert `role="dialog"`, `aria-modal="true"`, a non-empty
`aria-labelledby`, Escape close behavior, focus entry, and focus restoration.

- [ ] **Step 2: Run the targeted test and verify RED**

Run:

```powershell
npx playwright test tests/e2e/modal-consistency.spec.ts
```

Expected: FAIL because `BaseModal` and its semantic size attributes do not
exist.

- [ ] **Step 3: Commit the failing test**

```powershell
git add frontend/tests/e2e/modal-consistency.spec.ts
git commit -m "test: cover modal consistency and containment"
```

### Task 2: Create the shared modal primitive

**Files:**
- Create: `frontend/src/components/common/BaseModal.vue`

- [ ] **Step 1: Implement the public API and behavior**

Implement:

```ts
type ModalSize = 'small' | 'medium' | 'large'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  size?: ModalSize
  closeOnOverlay?: boolean
}>(), {
  size: 'small',
  closeOnOverlay: true,
})

const emit = defineEmits<{ close: [] }>()
```

Track the dialog element and the previously focused element. When `open`
becomes true, save `document.activeElement`, wait for `nextTick`, and focus the
dialog. On Escape emit `close`. When closed or unmounted, remove the key
listener and restore focus if the saved element remains connected.

- [ ] **Step 2: Implement the shared template**

Use `Teleport` to `body` and render:

```vue
<div v-if="open" class="modal-overlay" @click.self="handleOverlayClose">
  <section
    ref="dialogRef"
    class="modal-dialog"
    :class="`modal-dialog--${size}`"
    :data-modal-size="size"
    data-testid="base-modal"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    tabindex="-1"
  >
    <header class="modal-header">
      <h2 :id="titleId">{{ title }}</h2>
    </header>
    <div class="modal-body"><slot /></div>
    <footer v-if="$slots.actions" class="modal-actions">
      <slot name="actions" />
    </footer>
  </section>
</div>
```

- [ ] **Step 3: Implement the semantic sizing CSS**

The shared surface must use:

```css
.modal-dialog {
  --modal-width: 420px;
  box-sizing: border-box;
  width: min(var(--modal-width), calc(100vw - 32px));
  min-width: 0;
  max-height: calc(100dvh - 32px);
  display: flex;
  flex-direction: column;
}

.modal-dialog--medium { --modal-width: 500px; }
.modal-dialog--large {
  --modal-width: 800px;
  height: min(80dvh, calc(100dvh - 32px));
}
```

Centralize the existing background, padding, radius, shadow, overlay, heading,
body scrolling, and footer spacing in this component.

- [ ] **Step 4: Run the frontend build**

Run:

```powershell
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit the primitive**

```powershell
git add frontend/src/components/common/BaseModal.vue
git commit -m "feat: add responsive base modal"
```

### Task 3: Migrate compact dialogs

**Files:**
- Modify: `frontend/src/components/modals/CategoryModal.vue`
- Modify: `frontend/src/components/modals/MoveTabModal.vue`
- Modify: `frontend/src/components/modals/BatchMoveModal.vue`
- Modify: `frontend/src/components/modals/ConfirmModal.vue`
- Modify: `frontend/src/components/modals/PluginSettingsModal.vue`

- [ ] **Step 1: Replace duplicated shells**

Import `BaseModal` in each file and replace each overlay/surface/heading with:

```vue
<BaseModal
  :open="visibleState"
  :title="translatedTitle"
  size="small"
  @close="hideAction"
>
  <!-- existing fields or confirmation message -->
  <template #actions>
    <!-- existing buttons -->
  </template>
</BaseModal>
```

Give each form a stable ID and point its submit button at that form:

```vue
<form id="category-form" @submit.prevent="handleSave">...</form>
<button type="submit" form="category-form" class="btn primary">...</button>
```

Use `.modal-actions__leading` for the Confirm alternate action and remove the
two inline footer styles.

- [ ] **Step 2: Remove obsolete local shell CSS**

Delete component styles that set outer modal width, height, padding, radius,
shadow, or overlay behavior. Keep only feature-content styles such as
`.cover-input` and `.modal-content`.

- [ ] **Step 3: Run the build and targeted browser test**

Run:

```powershell
npm run build
npx playwright test tests/e2e/modal-consistency.spec.ts
```

Expected: the five small cases pass; remaining cases still fail because they
have not migrated.

- [ ] **Step 4: Commit the compact-dialog migration**

```powershell
git add frontend/src/components/modals
git commit -m "refactor: migrate compact dialogs to base modal"
```

### Task 4: Migrate medium and large dialogs

**Files:**
- Modify: `frontend/src/components/modals/EditTabModal.vue`
- Modify: `frontend/src/components/modals/KeyBindingModal.vue`
- Modify: `frontend/src/components/modals/CloudUploadModal.vue`
- Modify: `frontend/src/components/modals/WebDAVModal.vue`
- Modify: `frontend/src/components/modals/CloudPickerModal.vue`

- [ ] **Step 1: Migrate medium dialogs**

Use `size="medium"` for Edit Tab, Key Bindings, Cloud Upload, and WebDAV. Move
existing action buttons to the `actions` slot. Add form IDs and `form`
attributes where submission currently depends on nesting.

Rename content-specific `.modal-body` selectors so they cannot collide with the
shared body, for example:

```css
.cloud-upload-content { ... }
.key-binding-content { ... }
.webdav-form { ... }
```

- [ ] **Step 2: Migrate the Cloud Picker**

Use `size="large"`. Wrap its toolbar and file list in
`.cloud-picker-content`, preserving the current flex layout:

```css
.cloud-picker-content {
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
```

Move all action buttons to the shared footer and remove `.modal.large`.

- [ ] **Step 3: Remove the obsolete global modal shell**

Delete `.modal-overlay`, `.modal`, `.modal h2`, `.modal-actions`, and
`.confirm-modal` rules from `frontend/src/assets/style.css`. Preserve form and
confirm-message content rules.

- [ ] **Step 4: Run RED-to-GREEN verification**

Run:

```powershell
npm run build
npx playwright test tests/e2e/modal-consistency.spec.ts
```

Expected: PASS for all ten dialogs at desktop and mobile viewports.

- [ ] **Step 5: Commit the remaining migration**

```powershell
git add frontend/src/components/modals frontend/src/assets/style.css
git commit -m "refactor: unify all modal layouts and sizes"
```

### Task 5: Unify notification surfaces

**Files:**
- Modify: `frontend/src/assets/style.css`
- Modify: `frontend/src/components/common/Toast.vue`
- Modify: `frontend/src/components/common/SyncTaskToast.vue`
- Modify: `frontend/tests/e2e/modal-consistency.spec.ts`

- [ ] **Step 1: Extend the test with shared-notification assertions**

Read both component source files and assert each template contains
`notification-surface`. In the browser, append two representative elements
with `notification-surface`, `toast`, and `toast-mode` classes and assert their
computed padding, radius, border-left width, shadow, and maximum width match.

At 375px, insert a long message and assert the surface stays within the
24px-right/left notification margin.

- [ ] **Step 2: Run the notification test and verify RED**

Run:

```powershell
npx playwright test tests/e2e/modal-consistency.spec.ts -g notification
```

Expected: FAIL because the shared class and tokens do not exist.

- [ ] **Step 3: Add shared design tokens and surface class**

Define the approved `--notification-*` properties in `:root` and implement:

```css
.notification-surface {
  box-sizing: border-box;
  padding: var(--notification-padding-block) var(--notification-padding-inline);
  border: var(--notification-border-width) solid var(--border);
  border-left-width: var(--notification-accent-width);
  border-radius: var(--notification-radius);
  box-shadow: var(--notification-shadow);
  width: fit-content;
  max-width: min(var(--notification-max-width), calc(100vw - 48px));
}
```

Add `notification-surface` to ordinary toast and sync-task toast markup. Remove
their duplicate padding, radius, border, shadow, and width declarations while
retaining status colors, placement, transitions, and lifecycle behavior.

- [ ] **Step 4: Run targeted tests and build**

Run:

```powershell
npx playwright test tests/e2e/modal-consistency.spec.ts
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit notification unification**

```powershell
git add frontend/src/assets/style.css frontend/src/components/common frontend/tests/e2e/modal-consistency.spec.ts
git commit -m "refactor: unify notification surfaces"
```

### Task 6: Final verification

**Files:**
- Verify all modified files.

- [ ] **Step 1: Check formatting and accidental changes**

Run:

```powershell
git diff --check
git status --short
```

Expected: no whitespace errors and only intentional files.

- [ ] **Step 2: Run frontend verification**

Run:

```powershell
npm run build
npx playwright test
```

Expected: PASS.

- [ ] **Step 3: Run backend verification**

Run:

```powershell
go test ./pkg/sync -count=1
go test ./...
```

Expected: PASS.
