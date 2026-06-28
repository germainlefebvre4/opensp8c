## Context

L'ExplorePanel existant s'affiche comme un slot latéral droit de 420px dans `KanbanPage` (layout `flex-row`). Il partage le même mécanisme d'état `activePanel` que le `DetailPanel`. Le backend lance un subprocess `claude` long-lived, mais sans message initial : la session démarre comme un chat Claude générique. Si la WebSocket se déconnecte, l'historique de la conversation est perdu car il n'existe pas de buffer côté serveur.

## Goals / Non-Goals

**Goals:**
- Déplacer l'ExplorePanel vers un bottom panel fixe, sous les colonnes Kanban
- Permettre le redimensionnement vertical par drag
- Auto-injecter `/opsx:explore <changeName>` au démarrage de session
- Conserver l'historique en mémoire serveur et rejouer sur reconnexion WebSocket

**Non-Goals:**
- Persister l'historique des sessions sur disque
- Déplacer le DetailPanel (reste en slot latéral droit)
- Supporter le redimensionnement horizontal du bottom panel

## Decisions

### D1 — Layout KanbanPage : flex-col au lieu de flex-row

`KanbanPage` passe d'un layout `flex-row` à `flex-col` :
- Ligne du haut (`flex-1`) : colonnes Kanban + slot détail 420px (inchangé pour DetailPanel)
- Ligne du bas (conditionnelle) : `ExploreBottomPanel` à hauteur fixée par état

```
flex-col
├── flex-row flex-1         ← colonnes + DetailPanel (inchangé)
│   ├── colonnes flex-1
│   └── DetailPanel 420px (si ouvert)
└── ExploreBottomPanel      ← nouveau, hauteur variable
    ├── drag handle (4px)
    ├── header + close
    └── chat (ExplorePanel existant)
```

Alternatives considérées :
- Overlay full-screen : cache le Kanban, réduit la valeur du contexte visuel pendant la conversation
- Slot latéral élargi : 3 panneaux = espace horizontal étouffant sur petits écrans

### D2 — Drag-to-resize : événements souris sur window

Le drag handle est un div de 4px en haut du bottom panel avec `cursor-row-resize`. Sur `mousedown`, on attache `mousemove` et `mouseup` à `window`. La hauteur est calculée comme `clientHeight - e.clientY` à chaque `mousemove`, clampée entre 200px et 70% de la hauteur fenêtre. L'état `panelHeight` est local à `KanbanPage`.

Alternatives considérées :
- Bibliothèque de resize (react-resizable, etc.) : dépendance externe non justifiée pour un seul composant
- CSS resize : ne permet pas de contrôle de min/max ni d'animation custom

### D3 — Auto-injection du premier message : côté backend dans Manager.Start

Au démarrage d'une session (`Manager.Start`), après le spawn du subprocess, on envoie immédiatement `{"type":"user","message":{"role":"user","content":"/opsx:explore <changeName>"}}\n` sur stdin. Ce message n'est pas visible comme message utilisateur dans l'UI (il n'est pas émis par le WebSocket) — le frontend voit seulement la réponse de Claude.

Alternatives considérées :
- Frontend auto-send au `onopen` : risque de double-envoi sur reconnexion si le frontend ne détecte pas que la session est déjà initiée
- `--append-system-prompt` : n'invoque pas le skill, seulement une instruction texte sans accès aux outils OpenSpec

### D4 — Buffer messages : goroutine dédiée + fan-out

La `Session` acquiert un champ `messages [][]byte` protégé par un mutex. Une goroutine dédiée dans `Manager.Start` lit en continu le stdout du subprocess, ajoute chaque ligne au buffer, puis la pousse dans un channel (`outCh chan []byte`). Le handler WebSocket consomme ce channel. Sur reconnexion, il rejoue d'abord le buffer entier (`messages`), puis consomme le channel en live.

```
subprocess stdout
       ↓
  goroutine buffer  →  messages[]  (persistant)
       ↓
    outCh chan
       ↓
  handler WS (reconnectable)
```

Alternatives considérées :
- Lecture stdout directement dans le handler WS : non reconnectable, stdout est un io.ReadCloser consommé une seule fois
- Persister sur disque : overhead non justifié, lifetime de session = durée du subprocess (max 30 min)

## Risks / Trade-offs

- **Buffer illimité** → Pour des sessions très longues (>100 messages), la mémoire croît. Mitigation : cap à 500 messages (drop les plus anciens) acceptable pour l'usage explore.
- **Message initial non affiché** → Si Claude ne reconnaît pas `/opsx:explore` en stream-json, la session démarre silencieusement sans guide. Mitigation : le handler peut renvoyer une erreur visible si le subprocess ne produit pas de réponse dans les 5 secondes.
- **Drag sur mobile** → Les touch events ne sont pas couverts dans D2. Mitigation : hors scope, l'app cible desktop.

## Open Questions

- Aucune. Les décisions de l'exploration (`/opsx:explore`) ont résolu tous les points ouverts.
