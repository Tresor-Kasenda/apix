Voici la liste complète des fonctionnalités pour la v0.1.0 d'apix, organisées par priorité.

---

## Tier 1 — Core (sans ça, l'outil ne sert à rien)

**Requêtes HTTP de base**
- `apix get <path>` — envoie un GET
- `apix post <path>` — envoie un POST avec body
- `apix put <path>` — envoie un PUT avec body
- `apix patch <path>` — envoie un PATCH avec body
- `apix delete <path>` — envoie un DELETE
- `apix head <path>` — envoie un HEAD (retourne uniquement les headers)
- `apix options <path>` — envoie un OPTIONS (utile pour CORS debugging)

**Options de body**
- `--data` / `-d` — body JSON inline : `apix post /users -d '{"name":"John"}'`
- `--file` / `-f` — body depuis un fichier : `apix post /users -f payload.json`
- `--form` — envoie en `multipart/form-data` : `apix post /upload --form file=@photo.jpg`
- `--urlencoded` — envoie en `application/x-www-form-urlencoded`

**Options de requête**
- `--header` / `-H` — ajoute un header (repeatable) : `apix get /users -H "X-Custom: value"`
- `--query` / `-q` — query parameters : `apix get /users -q "page=2&limit=10"`
- `--timeout` / `-t` — timeout en secondes (override config)
- `--no-follow` — ne pas suivre les redirections

**Options d'affichage**
- `--verbose` / `-v` — affiche la requête complète envoyée + réponse complète
- `--raw` — affiche le body brut sans formatting
- `--headers-only` — affiche uniquement les headers de réponse
- `--body-only` — affiche uniquement le body (utile pour piping)
- `--silent` / `-s` — aucun output sauf le body (pour scripts)
- `--output` / `-o` — sauvegarde la réponse dans un fichier

---

## Tier 2 — Project Intelligence (ce qui différencie apix de curl)

**Initialisation de projet**
- `apix init` — crée `apix.yaml`, `requests/`, `env/`, `.apix/`
- Demande interactivement : nom du projet, base URL, type d'auth
- Génère un `env/dev.yaml` par défaut
- Ajoute `.apix/` au `.gitignore`

**Configuration projet (apix.yaml)**
- `project` — nom du projet
- `base_url` — URL de base pour toutes les requêtes
- `headers` — headers par défaut appliqués à chaque requête
- `timeout` — timeout global en secondes
- `auth` — configuration d'authentification

**Gestion des environnements**
- `apix env use <name>` — active un environnement
- `apix env list` — liste tous les environnements disponibles
- `apix env show` — affiche l'env actif avec toutes ses variables
- `apix env create <name>` — crée un nouvel environnement
- `apix env delete <name>` — supprime un environnement (avec confirmation)
- `apix env copy <source> <dest>` — duplique un env
- Chaque env override : base_url, headers, variables, auth

**Variables**
- `${VAR}` — résolue depuis l'env actif ou le flag `--var`
- `${TOKEN}` — token capturé automatiquement
- `${TIMESTAMP}` — timestamp Unix actuel
- `${ISO_DATE}` — date ISO 8601 actuelle
- `${UUID}` — UUID v4 généré à chaque exécution
- `${RANDOM}` — nombre aléatoire
- `${RANDOM_EMAIL}` — email aléatoire pour les tests
- `--var` / `-V` — override de variable en ligne : `apix get /users/${id} -V id=5`
- Résolution dans : path, headers, body, query params

---

## Tier 3 — Auth Intelligence (la killer feature)

**Auto auth capture**
- Après chaque POST, parse le body JSON
- Cherche le token au `token_path` défini dans config (ex: `data.token`, `access_token`, `key`)
- Si trouvé, sauvegarde dans `.apix/token`
- Affiche `✓ Token captured and saved`
- Toutes les requêtes suivantes injectent ce token automatiquement

**Types d'auth supportés**
- `bearer` — `Authorization: Bearer <token>`
- `basic` — `Authorization: Basic <base64(user:pass)>`
- `api_key` — header custom (ex: `X-API-Key: <key>`)
- `custom` — header et format entièrement personnalisables

**Configuration auth dans apix.yaml**
```yaml
auth:
  type: bearer
  token_path: data.token        # où trouver le token
  header_name: Authorization    # quel header utiliser
  header_format: "Bearer ${TOKEN}"  # format du header
  login_request: login          # requête sauvegardée pour auto-login
```

**Auto refresh**
- Détecte les réponses 401 Unauthorized
- Si `login_request` est configuré, relance automatiquement le login
- Recapture le token et re-tente la requête originale
- Affiche `✓ Token expired, re-authenticated automatically`

---

## Tier 4 — Collections (organisation des requêtes)

**Sauvegarder des requêtes**
- `apix save <name>` — sauvegarde la dernière requête exécutée
- `apix save <name> --from-last` — explicitement depuis la dernière requête
- Sauvegarde dans `requests/<name>.yaml`

**Format d'une requête sauvegardée**
```yaml
name: create-user
method: POST
path: /users
headers:
  X-Custom: value
body:
  name: John Doe
  email: john@example.com
  role: admin
```

**Exécuter des requêtes sauvegardées**
- `apix run <name>` — exécute une requête sauvegardée
- `apix run <name> --var email=new@test.com` — override de variables
- `apix run <name> --env staging` — override d'environnement pour cette exécution

**Lister et gérer les requêtes**
- `apix list` — liste toutes les requêtes sauvegardées
- `apix show <name>` — affiche le contenu d'une requête sans l'exécuter
- `apix delete <name>` — supprime une requête sauvegardée
- `apix rename <old> <new>` — renomme une requête

**Chain requests (exécution séquentielle)**
- `apix chain <req1> <req2> <req3>` — exécute plusieurs requêtes en séquence
- Les variables capturées se propagent d'une requête à la suivante
- Capture de variables entre requêtes :

```yaml
# requests/login.yaml
name: login
method: POST
path: /login
body:
  email: ${ADMIN_EMAIL}
  password: ${ADMIN_PASS}
capture:
  TOKEN: data.token
  USER_ID: data.user.id

# requests/get-profile.yaml
name: get-profile
method: GET
path: /users/${USER_ID}
# USER_ID est automatiquement disponible depuis le login
```

---

## Tier 5 — Testing et validation

**Assertions dans les requêtes**
```yaml
name: test-login
method: POST
path: /login
body:
  email: test@test.com
  password: "123456"
expect:
  status: 200
  body.data.token: exists
  body.data.user.email: "test@test.com"
  body.data.user.id: is_number
  headers.content-type: contains "application/json"
  response_time: lt 500ms
```

**Opérateurs d'assertion disponibles**
- `exists` — le champ existe
- `not_exists` — le champ n'existe pas
- `eq` / `=` — égalité exacte
- `neq` / `!=` — différent de
- `contains` — contient la sous-chaîne
- `starts_with` — commence par
- `ends_with` — finit par
- `matches` — regex match
- `is_number` — est un nombre
- `is_string` — est une string
- `is_array` — est un tableau
- `is_bool` — est un booléen
- `is_null` — est null
- `gt`, `gte`, `lt`, `lte` — comparaisons numériques
- `length` — longueur d'un array ou string

**Mode test**
- `apix test` — exécute toutes les requêtes qui ont un bloc `expect`
- `apix test <name>` — exécute un test spécifique
- `apix test --dir tests/` — exécute tous les tests d'un dossier
- Affiche un résumé : passed, failed, total, durée
- Retourne exit code 0 si tout passe, 1 sinon (pour CI/CD)

**Output du mode test**
```
Running 5 tests...

  ✓ login                    200 OK    (142ms)
  ✓ get-users                200 OK    (89ms)
  ✗ create-user              422 Error (67ms)
    → expected status 201, got 422
    → expected body.id to exist, but was missing
  ✓ update-user              200 OK    (95ms)
  ✓ delete-user              204 OK    (71ms)

Results: 4 passed, 1 failed, 5 total (464ms)
```

---

## Tier 6 — Developer Experience

**Pretty output**
- JSON indenté et coloré par défaut
- Status code coloré : vert 2xx, jaune 3xx, rouge 4xx/5xx
- Temps de réponse affiché
- Taille de la réponse affichée

**Historique**
- `apix history` — affiche les 20 dernières requêtes exécutées
- `apix history --limit 50` — plus de résultats
- `apix history --clear` — efface l'historique
- Chaque entrée : méthode, path, status, durée, timestamp

**Informations utilitaires**
- `apix --version` / `-v` — version de l'outil
- `apix --help` — aide globale
- `apix <command> --help` — aide par commande
- `apix config show` — affiche la config active (config + env merged)
- `apix completion bash/zsh/fish` — génère l'autocomplétion shell

**Import depuis d'autres outils**
- `apix import postman <file.json>` — importe une collection Postman
- `apix import insomnia <file.json>` — importe depuis Insomnia
- `apix import curl <"curl command">` — convertit une commande curl en requête apix

**Export**
- `apix export curl <name>` — exporte une requête en commande curl
- `apix export postman` — exporte toutes les requêtes en collection Postman

---

## Tier 7 — Avancé (v0.2.0+, mais prévoir l'architecture maintenant)

**Watch mode**
- `apix watch <name>` — relance la requête à chaque modification du fichier YAML
- `apix watch <name> --interval 5s` — relance toutes les 5 secondes
- Utile pour développer et tester en temps réel

**Hooks pre/post request**
```yaml
name: create-user
method: POST
path: /users
pre_request:
  - run: login          # exécute login avant si pas de token
post_request:
  - capture:
      USER_ID: body.id
```

**Retry et resilience**
- `--retry 3` — réessaie 3 fois en cas d'erreur réseau
- `--retry-delay 2s` — délai entre les tentatives
- Backoff exponentiel automatique

**Proxy support**
- `--proxy http://localhost:8080` — route via un proxy
- Utile pour debugging avec Charles/mitmproxy

**TLS/SSL**
- `--insecure` / `-k` — ignore les erreurs de certificat SSL
- `--cert <file>` — certificat client
- `--key <file>` — clé privée client

**Cookie jar**
- Sauvegarde automatique des cookies entre requêtes
- `--no-cookies` — désactive le cookie jar
- Utile pour APIs qui utilisent des sessions

---

## Résumé par priorité d'implémentation

| Priorité | Catégorie | Nb de features | Effort |
|---|---|---|---|
| **P0** | Requêtes HTTP de base | 7 méthodes + flags | 2h |
| **P0** | Pretty output coloré | Status, headers, body, time | 1h |
| **P0** | Config projet (apix.yaml) | Chargement + merge | 1h |
| **P1** | Init + environnements | init, env use/list/show/create | 1.5h |
| **P1** | Variables | Résolution ${VAR} partout | 30min |
| **P1** | Auto auth capture | Détection token + injection auto | 45min |
| **P2** | Collections save/run | save, run, list, show, delete | 1h |
| **P2** | Chain requests + capture | Exécution séquentielle + propagation vars | 1h |
| **P3** | Test assertions | expect block + apix test + exit codes | 2h |
| **P3** | Historique | history + clear | 30min |
| **P4** | Import/Export | Postman, Insomnia, curl | 2h |
| **P4** | Auto refresh 401 | Détection 401 + re-login auto | 1h |
| **P5** | Watch mode, hooks, retry, proxy, TLS, cookies | Features avancées | v0.2.0+ |

**P0 + P1 = MVP minimal** (~6h) — suffisant pour une première release et première vidéo YouTube.

**P0 + P1 + P2 + P3 = v0.1.0 complète** (~12h) — suffisant pour que les gens adoptent l'outil au quotidien.