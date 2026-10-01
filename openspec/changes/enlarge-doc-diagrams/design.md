# Design

## Context

`MermaidDiagram.tsx` rend un bloc ```mermaid``` via `mermaid.render`, stocke la chaîne SVG résultante dans un état local et l'injecte avec `dangerouslySetInnerHTML` dans un `<div className="mermaid-diagram">`. Le composant n'est atteint que depuis le rendu Markdown de `DocumentationPanel.tsx` (`CodeRenderer`). Son repli sur échec est un `<pre><code>` brut.

Mermaid pose sur le `<svg>` produit un attribut `width="100%"` et un style inline `max-width: <largeur naturelle>px`. Un simple `width: 100%` en CSS ne suffit donc pas à agrandir un petit diagramme : le `max-width` inline doit être surchargé.

`@radix-ui/react-dialog` est déjà utilisé (`components/ui/ConfirmDialog.tsx`, avec `Dialog.Portal`, `Dialog.Overlay`, `Dialog.Content`) et fournit piège de focus, fermeture par Échap et par clic extérieur, restauration du focus.

Le change en cours `docs-toc-wide-reading` modifie aussi le style de `.mermaid-diagram` (`overflow-x: auto`, `svg { max-width: 100%; height: auto }`) et la mise en page de `DocumentationPanel`. Ce change-ci ne dépend pas de l'ordre d'intégration, mais les deux touchent `MermaidDiagram.tsx` : le style en ligne ne doit pas être redéfini ici, seulement celui de la modale.

Voir proposal.md pour la motivation et specs/spec-documentation-view/spec.md pour le comportement attendu.

## Goals / Non-Goals

**Goals:**
- Agrandissement via une lightbox, sans nouvelle dépendance.
- Aucun nouveau rendu Mermaid : le SVG déjà produit est réutilisé.
- Localiser le changement dans `MermaidDiagram.tsx` : `DocumentationPanel` et le pipeline Markdown restent inchangés.

**Non-Goals:**
- Zoom molette / boutons et déplacement (pan) dans la modale.
- Composant de lightbox générique réutilisable pour d'autres contenus.
- Export ou téléchargement du SVG.

## Decisions

**1. Radix Dialog contrôlé, local à `MermaidDiagram`.**
Un état `open` local par instance, un `Dialog.Root open onOpenChange` rendu dans un `Dialog.Portal`. Le portail évite que la modale soit rognée par les conteneurs `overflow` de la zone de documentation (`ScrollArea`) et que ses clics se propagent au Markdown.
*Alternatives* : overlay `fixed` fait main comme `DeleteChangeDialog` (il faudrait réimplémenter focus, Échap et accessibilité) ; expansion en place (rejetée avec l'utilisateur : la lisibilité reste limitée par le panneau).

**2. Le déclencheur est un `<button type="button">` enveloppant le SVG en ligne.**
Il apporte focus clavier, Entrée/Espace et rôle gratuitement, plutôt qu'un `onClick` sur le `div`. Classes : `cursor-zoom-in`, anneau de focus visible, `aria-label` traduit (clé `docs.diagram.enlarge`). Le `div.mermaid-diagram` et son `dangerouslySetInnerHTML` restent le contenu du bouton, donc le style posé par `docs-toc-wide-reading` continue de s'appliquer.
*Alternative* : `role="button"` + `tabIndex` sur le `div` (équivalent mais à gérer à la main).

**3. Le SVG de la modale est le même markup, réinjecté, avec un style d'échelle dédié.**
Le même `svg` (chaîne de l'état) est injecté dans `Dialog.Content`, sans second appel à `mermaid.render` (coûteux, et il réutiliserait un id déjà présent dans le DOM, source de collision `id` SVG entre les deux copies). Dans la modale, la copie est dimensionnée par une classe CSS dédiée : `svg { width: 100%; height: 100%; max-width: none !important; }` dans un conteneur borné à ~90vw × 90vh, `preserveAspectRatio` par défaut de Mermaid (`xMidYMid meet`) conservant les proportions. `!important` est nécessaire pour battre le `max-width` inline de Mermaid.
*Alternative* : re-render avec un nouvel id dans la modale (coût inutile, mêmes résultats) ; manipuler le DOM pour retirer l'attribut `style` du SVG (plus fragile que du CSS).
*Point d'attention* : deux copies du SVG coexistent dans le DOM pendant que la modale est ouverte. Les `id` internes de Mermaid (marqueurs de flèches, etc.) sont préfixés par `diagramId` et seront donc dupliqués ; comme les deux copies sont identiques, les références `url(#id)` restent valides quel que soit l'élément résolu.

**4. Repli et chargement inchangés.**
`failed` renvoie toujours `<pre><code>` sans bouton ; `svg === null` renvoie toujours `null`. Seule la branche « SVG disponible » est enveloppée.

**5. Libellés via i18n `specs`.**
Nouvelles clés sous `docs.diagram` (`enlarge`, `close`, `modalTitle`) en fr et en, dans l'espace de noms `specs` déjà utilisé par la documentation. `Dialog.Title` est rendu visuellement masqué (`sr-only`) pour satisfaire l'accessibilité Radix sans alourdir l'interface ; `aria-describedby={undefined}` évite l'avertissement sur la description absente.

## Risks / Trade-offs

- [`!important` sur `max-width` couplé à un détail de rendu de Mermaid] → limité à la classe de la modale ; un test de composant vérifie que le SVG de la modale n'est pas contraint, et un contrôle visuel sur `architecture.md` est prévu dans les tâches.
- [Les très gros diagrammes restent denses même agrandis, sans zoom] → assumé pour la v1 (non-but) ; l'ajout de zoom/pan est possible plus tard sans changer le contrat de clic/fermeture.
- [Conflit de merge avec `docs-toc-wide-reading` dans `MermaidDiagram.tsx`] → les modifications sont disjointes (ce change n'ajoute que l'enveloppe bouton, la modale et leurs classes) ; rebaser sur le change voisin s'il est intégré en premier.
- [Doublon d'`id` SVG pendant l'ouverture] → voir décision 3, sans impact visuel car les deux copies sont identiques.
