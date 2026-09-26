# apix — TODO

Liste de tout ce qui reste à faire : corrections, sécurité, fonctionnalités
manquantes, qualité, CI/release, documentation et contenu.

Chaque tâche indique **où** intervenir et **quand la considérer comme terminée**
(critère d'acceptation).

**Légende des priorités**

| Priorité | Signification |
|----------|---------------|
| 🔴 P0 | Bloquant ou bug visible par les utilisateurs : à faire avant la prochaine release |
| 🟠 P1 | Important : corrige une incohérence ou complète une fonctionnalité promise |
| 🟡 P2 | Amélioration notable de l'expérience ou de la robustesse |
| 🟢 P3 | Bonus, idées pour les versions suivantes |

---

## 1. Bugs et corrections

### 🔴 P0 — Chemin du module Go incohérent avec le dépôt
- **Problème** : `go.mod` déclare `github.com/Tresor-Kasend/apix`, mais le dépôt est `github.com/Tresor-Kasenda/apix`. Du coup `go install github.com/Tresor-Kasenda/apix/cmd/apix@latest` échoue.
- **Où** : `go.mod`, tous les imports `github.com/Tresor-Kasend/apix/...`, `Makefile` (`MODULE`).
- **Fait quand** : `go install github.com/Tresor-Kasenda/apix/cmd/apix@latest` fonctionne et le README documente cette méthode d'installation.

### 🔴 P0 — Le token capturé est partagé entre tous les environnements
- **Problème** : `.apix/token` est un fichier unique. Après `apix env use prod`, le token de `dev` est encore injecté, ce qui peut fuiter vers le mauvais serveur.
- **Où** : `internal/config/config.go` (`SaveToken`, `loadToken`).
- **Fait quand** : le token est stocké par environnement (`.apix/tokens/<env>`), avec une migration de l'ancien fichier et un test couvrant le changement d'environnement.

### 🟠 P1 — Viper met les noms d'en-têtes en minuscules
- **Problème** : dans `headers:`, `Content-Type` devient `content-type`, ce qui est visible dans `apix config show`. Certains serveurs non conformes y sont sensibles.
- **Où** : `internal/config/config.go` (`loadBase`). Option : décoder `apix.yaml` avec `yaml.v3` au lieu de viper, qui n'apporte rien ici.
- **Fait quand** : la casse des clés `headers` et `variables` est préservée, avec un test.

### 🟠 P1 — Un body YAML structuré est refusé
- **Problème** : `SavedRequest.Body` est un `string`. Un body écrit en YAML (`body: {name: John}`), pourtant prévu dans la spec, provoque une erreur de parsing.
- **Où** : `internal/request/collection.go` (type `SavedRequest`), `internal/cli/root.go` (`buildRequestBody`).
- **Fait quand** : `body` accepte une chaîne **ou** une map/liste, sérialisée en JSON à l'envoi, avec des variables résolues à l'intérieur des valeurs.

### 🟠 P1 — Les routes `web.php` de Laravel héritent du préfixe `/api`
- **Problème** : `apix init` propose `http://localhost:8000/api`, mais les routes de `routes/web.php` ne sont pas servies sous `/api`.
- **Où** : `internal/detect/patterns.go` et `internal/cli/init_cmd.go`.
- **Fait quand** : seules les routes de `routes/api.php` sont enregistrées, ou bien les routes web sont préfixées par une URL absolue ou ignorées, avec un test.

### 🟠 P1 — Collisions de noms entre routes détectées
- **Problème** : deux routes différentes peuvent donner le même nom de fichier (ex. `GET /users/{id}` et `GET /users/:id` dans deux fichiers). La seconde écrase la première sans avertissement.
- **Où** : `internal/cli/init_cmd.go` (`saveDetectedRoutes`).
- **Fait quand** : on réutilise `uniqueRequestName` (déjà utilisé par l'import) et le nombre de routes affiché correspond au nombre de fichiers créés.

### 🟡 P2 — Faux positifs dans la détection de routes par regex
- **Problème** : les motifs Rails, Sinatra, Yii et Phoenix capturent n'importe quelle chaîne `get '/...'`, y compris dans les tests, les specs et les commentaires.
- **Où** : `internal/detect/patterns.go`, `internal/detect/scanner.go`.
- **Fait quand** : les dossiers `test/`, `tests/`, `spec/` et `__tests__/` sont ignorés, les lignes commentées sont exclues, et un test par framework vérifie l'absence de faux positif.

### 🟡 P2 — Le retry automatique rejoue des requêtes non idempotentes
- **Problème** : `--retry` relance aussi `POST` et `PATCH` sur une réponse 5xx, ce qui peut créer des doublons côté serveur.
- **Où** : `internal/http/client.go`.
- **Fait quand** : par défaut, seuls `GET`, `HEAD`, `OPTIONS`, `PUT` et `DELETE` sont rejoués sur 5xx (les erreurs réseau restent rejouées), et une option `--retry-all` permet de forcer le comportement actuel.

### 🟡 P2 — Code dupliqué pour la validation du body
- **Où** : `internal/cli/root.go`. `validateBodyModes` et le début de `buildRequestBody` comptent les modes de body de la même façon.
- **Fait quand** : une seule fonction assure ce contrôle.

### 🟡 P2 — `apix delete` a un double sens
- **Problème** : `apix delete <path>` envoie une requête HTTP DELETE, mais `apix delete <name> --saved` supprime un fichier. Un oubli du flag envoie une vraie requête DELETE.
- **Fait quand** : une commande dédiée `apix rm <name>` (ou `apix requests delete`) existe, et `--saved` est déprécié avec un message d'avertissement.

---

## 2. Sécurité

### 🔴 P0 — `apix config show` affiche les secrets en clair
- **Où** : `internal/cli/config_show.go`. Les champs `auth.token`, `auth.password` et `auth.api_key` sont imprimés tels quels.
- **Fait quand** : ces valeurs sont masquées (`abc1…****`) par défaut, un flag `--reveal` les affiche, et un test le vérifie.

### 🟠 P1 — L'historique enregistre les URL résolues
- **Problème** : `.apix/history.jsonl` stocke l'URL finale, donc des secrets passés en query (`?api_key=${KEY}`) y apparaissent. Le fichier est créé en `0644`.
- **Où** : `internal/history/store.go`, `internal/cli/root.go`.
- **Fait quand** : on stocke le chemin **non résolu** ou on masque les paramètres sensibles (`token`, `key`, `secret`, `password`…), et les fichiers de `.apix/` sont créés en `0600`.

### 🟡 P2 — `--insecure` sans avertissement
- **Fait quand** : un avertissement jaune s'affiche quand la validation TLS est désactivée (sauf en `--silent`).

### 🟡 P2 — Téléchargement de spec OpenAPI
- **Où** : `internal/interop/openapi/import.go` (`readSource`).
- **Fait quand** : le téléchargement respecte `--proxy`, `--insecure` et les en-têtes d'auth du projet (specs protégées), et la taille maximale est documentée.

---

## 3. Fonctionnalités prévues mais pas encore implémentées

(reprises de l'ancienne spec v0.1.0)

### 🟠 P1 — Opérateurs d'assertion manquants
- **Où** : `internal/tester/assertions.go`.
- **À ajouter** : `not_exists`, `neq`, `starts_with`, `ends_with`, `matches` (regex).
- **À ajouter aussi** : la syntaxe courte de la spec (`body.data.token: exists`, `headers.content-type: contains "application/json"`, `response_time: lt 500ms`).
- **Fait quand** : chaque opérateur a son test et la liste du README est à jour.

### 🟠 P1 — Variables intégrées manquantes
- **Où** : `internal/request/variables.go` (`BuildVariableMap`).
- **À ajouter** : `${ISO_DATE}` (ISO 8601) et `${RANDOM_EMAIL}` (ex. `user-3f9a2c1b@example.com`).
- **Fait quand** : les variables sont documentées dans le tableau du README et testées.

### 🟡 P2 — Hooks : `pre_request` conditionnel
- **Spec** : « exécute login avant si pas de token ».
- **Fait quand** : un hook peut porter `if_missing: TOKEN` (ou une syntaxe équivalente) et n'est exécuté que si la variable est absente.

### 🟡 P2 — Export Insomnia et OpenAPI
- **Fait quand** : `apix export insomnia` et `apix export openapi` produisent des fichiers valides qu'on peut réimporter (test aller-retour).

---

## 4. Universalité : prochaines étapes

### 🟠 P1 — Import OpenAPI : couverture incomplète
- **Où** : `internal/interop/openapi/import.go`.
- **À faire** :
  - [ ] `$ref` vers des fichiers externes (`./schemas/user.yaml#/User`)
  - [ ] bodies `multipart/form-data` et `application/x-www-form-urlencoded`
  - [ ] schémas de sécurité (`components.securitySchemes`) → proposer la config `auth` adaptée
  - [ ] option `--tag <tag>` pour n'importer qu'une partie de l'API
  - [ ] option `--overwrite` / `--skip-existing` pour la réimportation (aujourd'hui des doublons `-2` sont créés)
  - [ ] ranger les requêtes dans des sous-dossiers par tag (`requests/users/…`)
- **Fait quand** : chaque point a son test, et réimporter une spec mise à jour ne duplique rien.

### 🟠 P1 — Sous-dossiers dans `requests/`
- **Problème** : `ListSaved`, `Load` et `apix test` ne lisent que le premier niveau. Les grandes APIs ont besoin d'une arborescence.
- **Fait quand** : `apix run users/create` fonctionne, `apix list` affiche l'arbre, et `apix test users/` lance un sous-ensemble.

### 🟡 P2 — Détection sans spec en mode « serveur lancé »
- **Idée** : `apix init --probe` interroge les URLs de spec connues (`/openapi.json`, `/api-json`, `/v3/api-docs`, `/swagger.json`, `/docs/api.json`, `/api/documentation`) sur l'URL de base détectée, et importe la première qui répond.
- **Fait quand** : cela fonctionne sur FastAPI, NestJS et Spring, et échoue proprement si le serveur est éteint.

### 🟡 P2 — Règles de détection définies par l'utilisateur
- **Idée** : permettre de déclarer un framework maison dans `apix.yaml` (fichiers marqueurs, extensions, regex avec les groupes `method` et `path`).
- **Fait quand** : un exemple est documenté et testé.

### 🟡 P2 — Autres frameworks à détecter
- [ ] Next.js / Nuxt (routes API basées sur les fichiers : `pages/api`, `app/api/**/route.ts`, `server/api`)
- [ ] Express Router monté avec préfixe (`app.use('/api', router)`)
- [ ] Litestar / Starlette, Sanic (Python)
- [ ] Laravel : lire `php artisan route:list --json` s'il est disponible (plus fiable que les regex)
- [ ] Django : lire les `include()` pour reconstituer les préfixes

### 🟢 P3 — Autres formats d'import
- [ ] Fichiers `.http` / `.rest` (JetBrains HTTP Client, extension VS Code REST Client)
- [ ] Collections Bruno (`.bru`)
- [ ] HAR (export des DevTools du navigateur)

### 🟢 P3 — Autres protocoles
- [ ] GraphQL (`apix gql <query-file>`, variables, introspection)
- [ ] WebSocket / SSE (lecture de flux)
- [ ] gRPC (via réflexion)

---

## 5. Expérience développeur (DX)

### 🟠 P1 — Messages d'erreur plus guidants
- Si aucun `apix.yaml` n'est trouvé et que le chemin est relatif : l'URL par défaut `http://localhost:8000/api` est utilisée **silencieusement**. Il faut afficher un conseil : `apix init`, `--base-url` ou `APIX_BASE_URL`.
- Si une variable reste non résolue dans l'URL, un en-tête ou le body : avertir (`⚠ ${USER_ID} is not defined`) au lieu d'envoyer `${USER_ID}` littéralement.
- **Fait quand** : les deux cas ont un message et un test.

### 🟡 P2 — Autocomplétion dynamique
- **Problème** : cobra fournit `apix completion`, mais les noms de requêtes et d'environnements ne sont pas complétés.
- **Fait quand** : `ValidArgsFunction` est défini pour `run`, `show`, `rename`, `chain`, `test`, `watch`, `env use/show/delete/copy`, `export curl`.

### 🟡 P2 — Commande `apix doctor`
- **Idée** : afficher la racine du projet, l'environnement actif, les fichiers `.env` chargés, l'URL de base effective, la présence d'un token, la joignabilité du serveur et les variables non résolues dans les requêtes.

### 🟡 P2 — Sortie machine
- [ ] `--json` pour `list`, `history`, `env list` et `config show`
- [ ] `apix test --reporter junit|json|tap` et `--output report.xml` pour GitHub Actions / GitLab CI

### 🟢 P3 — Mode interactif (TUI)
- Une liste des requêtes filtrable, exécutée avec Entrée. Lib possible : `charmbracelet/bubbletea`.

### 🟢 P3 — Lancer des tests en parallèle
- **Fait quand** : `apix test --parallel 4` fonctionne et les chaînes à dépendances restent séquentielles.

---

## 6. Qualité et tests

- [ ] 🟠 P1 : tests unitaires pour `internal/config` (précédence des variables, fichiers dotenv manquants ou invalides, `dotenv: []`) et `internal/project` (`APIX_PROJECT_DIR`, chemins absolus dans `requests_dir`).
- [ ] 🟠 P1 : tests pour `internal/output` (couleurs désactivées si ce n'est pas un TTY, `NO_COLOR`).
- [ ] 🟡 P2 : test de bout en bout du binaire (build + `init` + `run` + `test` contre un serveur `httptest`), en complément des tests de package.
- [ ] 🟡 P2 : activer `golangci-lint` (une cible `make lint` existe déjà) avec une configuration `.golangci.yml` versionnée.
- [ ] 🟡 P2 : mesurer la couverture (`go test -coverprofile`) et viser plus de 70 % sur `request`, `config`, `detect`, `tester` et `interop`.
- [ ] 🟢 P3 : fuzzing de `dotenv.Parse`, `openapi.Parse` et `curl.ParseCommand` (`go test -fuzz`).

---

## 7. CI / release

- [ ] 🔴 P0 : ajouter un workflow `.github/workflows/ci.yml` (seul `release.yaml` existe) qui lance `go vet`, `go test ./...` et `golangci-lint` sur chaque PR, sous Linux, macOS et Windows.
- [ ] 🟠 P1 : vérifier que tout fonctionne sous **Windows** : séparateurs de chemins dans `project.Root`, `filepath.Glob`, détection TTY et couleurs, `install.sh` non applicable (documenter Scoop / winget ou l'archive `.zip`).
- [ ] 🟡 P2 : publier une image Docker (`ghcr.io/tresor-kasenda/apix`) pour la CI.
- [ ] 🟡 P2 : fournir une GitHub Action réutilisable (`uses: tresor-kasenda/apix-action@v1` avec `apix test`).
- [ ] 🟡 P2 : générer le CHANGELOG automatiquement à partir des commits conventionnels (goreleaser `changelog`).
- [ ] 🟢 P3 : paquets Scoop (Windows), AUR (Arch) et `.deb` / `.rpm` via goreleaser `nfpms`.

---

## 8. Documentation

- [ ] 🟠 P1 : le README (plus de 600 lignes) est trop long. Le découper en `docs/` (installation, configuration, variables, auth, tests, import/export, CI), avec un site généré (MkDocs Material ou Docusaurus).
- [ ] 🟠 P1 : ajouter un exemple complet par stack dans `examples/` : Laravel, Django/DRF, FastAPI, NestJS, Spring Boot, Go, avec `apix.yaml`, `requests/` et `env/` prêts à l'emploi.
- [ ] 🟡 P2 : ajouter un guide de migration depuis Postman / Insomnia / fichiers `.http`.
- [ ] 🟡 P2 : ajouter `CONTRIBUTING.md` (architecture des packages, comment ajouter un framework à la détection, conventions de commit).
- [ ] 🟡 P2 : générer la référence des commandes automatiquement (`cobra/doc` → `docs/commands/*.md`) pour éviter que le tableau du README se désynchronise.
- [ ] 🟢 P3 : ajouter un GIF ou une vidéo asciinema en haut du README.

---

## 9. Contenu (YouTube / LinkedIn)

- [ ] Vidéo « De zéro à des tests d'API en CI en 5 minutes » : `apix init --spec` → `apix test` → GitHub Actions.
- [ ] Série « apix avec votre stack » : un épisode court par framework (Laravel, FastAPI, NestJS, Spring).
- [ ] Post LinkedIn « Pourquoi versionner ses requêtes API avec Git plutôt que dans Postman ».
- [ ] Post technique sur l'architecture : découverte de la racine façon `git`, précédence des variables, import OpenAPI.

---

## ✅ Déjà fait (pour mémoire)

- [x] Requêtes HTTP (GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS), bodies JSON, fichier, multipart et urlencoded
- [x] Options d'affichage (`-v`, `--raw`, `--headers-only`, `--body-only`, `-s`, `-o`)
- [x] `apix init`, environnements (`use/list/show/create/delete/copy`), variables
- [x] Auth `bearer/basic/api_key/custom`, capture automatique du token, re-login sur 401
- [x] Requêtes sauvegardées (`save/run/list/show/rename/delete`), `chain` et `capture`
- [x] Assertions et `apix test` (code de sortie CI), historique, `config show`
- [x] Import Postman / Insomnia / curl, export curl / Postman
- [x] Watch mode, hooks `pre_request` / `post_request`, retry, proxy, TLS, cookie jar
- [x] **Universalité** : découverte de la racine du projet, `requests_dir` / `env_dir`, fichiers `.env`, `${VAR:-défaut}`, variables de l'OS, `APIX_ENV` / `APIX_BASE_URL` / `APIX_PROJECT_DIR`
- [x] Import OpenAPI 3 / Swagger 2 (fichier ou URL), `init` non interactif, détection monorepo, URL de base déduite du `.env`, plus de 30 frameworks
- [x] Correction : `env/dev.yaml` n'écrase plus le `base_url` choisi à l'init
