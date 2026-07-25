# UI implementation guidelines

Use these rules for every user-visible interface change in Seicho.

## Reuse before adding

- Prefer existing tokens and component rules over one-off CSS values.
- Add a token only when a value represents a reusable design decision.
- Keep colors, typography, control sizes, and spacing consistent across related components.
- Limit element-specific selectors to behavior or presentation unique to that element.

## CSS responsibilities

- Define shared design values in `web/tokens.css`.
- Define reusable controls and component states in `web/component-theme.css`.
- Define page layout and responsive behavior in `web/interface-theme.css`.
- Define light and dark appearance differences in `web/color-schemes.css`.
- Avoid duplicating an existing rule in another stylesheet.

## Controls

- Use the shared control sizes:
  - Standard button: 36px high.
  - Compact button: 34px high.
  - Icon button: 34px square.
- Use the button padding tokens instead of introducing new horizontal padding.
- Preserve a usable click target when the text or icon is visually subdued.
- Use primary styling for the main action, secondary styling for alternatives, and destructive styling only for irreversible actions.
- Every interactive control must have appropriate hover, keyboard-focus, disabled, and loading states when applicable.
- Do not rely on color alone to communicate state or meaning.

## Typography and color

- Use the font-size variables in `web/tokens.css` according to the content role.
- Use the shared monospace font for multiline text editing, with font size selected according to the editor's content role.
- Keep the Markdown editor and Preview typography aligned by default. Share their font family, font size, line height, and content padding instead of assigning different values based on purpose.
- Introduce a Markdown–Preview difference only when the difference is necessary for content rendering or interaction, not merely because one area is editable and the other is a preview.
- Prefer weight and color changes over arbitrary font-size changes when adjusting visual emphasis.
- Use theme variables instead of fixed colors when a value should change between light and dark appearances.
- Render all placeholder text with the shared placeholder color, regular font weight, and normal letter spacing so its emphasis remains consistent; inherit font family and size from the field's content role.
- Check both light and dark themes after changing a shared component.

## Layout and responsive behavior

- Keep primary actions near the content they affect and separate destructive actions to reduce accidental use.
- Preserve the established visual hierarchy in the header, sidebar, editor toolbar, and dialogs.
- Check narrow-screen behavior when changing the header, sidebar, dialogs, menus, or editor toolbar.
- Avoid text wrapping that produces isolated words or unnatural line breaks in controls and dialogs.

## Language and accessibility

- Add Japanese and English strings together and keep their translation keys aligned.
- Give icon-only controls localized accessible names and visible focus indicators.
- Disable repeated submission while an asynchronous action is running and restore controls after success or failure.

## Verification checklist

- Confirm that an existing token or component rule cannot cover the change before adding CSS.
- Check normal, hover, focus, disabled, loading, error, and destructive states that apply.
- Check light and dark appearances.
- Check the narrow-screen layout when the affected area is responsive.
- Add or update a regression test for testable behavior.
- Follow the verification and changelog rules in `AGENTS.md`.
