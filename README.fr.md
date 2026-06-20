# Outil de récupération d’historique QQ Zone (qzone-history)

> **🌐 语言 / Language / Langue :** [简体中文](README.md) · [English](README.en.md) · [**Français**](README.fr.md)

[![Version](https://img.shields.io/badge/version-v0.0.1-brightgreen)](version/version.go)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows&logoColor=white)](#compilation-depuis-les-sources)
[![GitHub](https://img.shields.io/badge/GitHub-ZHChen2000/qzone--history-181717?style=flat&logo=github)](https://github.com/ZHChen2000/qzone-history)

**Auteur : [ZHChen](https://github.com/ZHChen2000)** · **Contact : QQ 1415094395**

Récupérez autant que possible les **publications et messages supprimés** depuis les flux « Activité », les API de publications et le mur de messages de QQ Zone, puis exportez-les en JSON local et pages HTML de consultation.

> Réservé à la sauvegarde personnelle de **votre propre** espace QQ Zone. Respectez les conditions d’utilisation de Tencent.

---

## Aperçu de l’interface

Après un double-clic sur `qzone-history-gui.exe`, le navigateur ouvre la console Web. Connectez-vous par QR code pour lancer la récupération :

![Vue d’ensemble de la console](docs/images/gui-overview.png)

Journaux en direct et progression :

![Journaux et progression](docs/images/gui-logs.png)

Une fois terminé, consultez publications, messages et activités sur une page HTML locale :

![Page de consultation des résultats](docs/images/viewer-result.png)

---

## Fonctionnalités

- Connexion par QR code QQ (flux officiel ; cookies stockés localement)
- Console Web : journaux, progression, arrêt / fermeture du processus
- Max Offset recommandé selon l’année cible ; réglage manuel du balayage profond
- Récupération des publications non supprimées ; reconstruction des supprimées via l’activité
- API du mur de messages ; repli sur reconstruction depuis l’activité
- Export `{QQ}_export.json`, `{QQ}_activities.json`, `{QQ}_view.html`

## Démarrage rapide

**Pas envie de compiler ?** Double-cliquez sur `qzone-history-gui.exe` et suivez [quickStart.md](./quickStart.md).

Étapes détaillées, tableau Offset, durées et FAQ → **[quickStart.md](./quickStart.md)** (en chinois)

## Structure du projet

```
qzone-history/
├── qzone-history-gui.exe   # Binaire Windows précompilé (sans fenêtre console)
├── docs/images/            # Captures pour la documentation
├── cmd/                    # Points d’entrée et outils de débogage
├── internal/               # Logique métier, GUI, client API
├── pkg/                    # Export, chemins, journaux, etc.
├── config/                 # Configuration par défaut
├── version/                # Version et auteur
├── go.mod
├── README.md
└── quickStart.md
```

## Compilation depuis les sources

Nécessite [Go 1.21+](https://go.dev/dl/).

```powershell
# Sans fenêtre console (recommandé pour la distribution)
go build -ldflags="-H windowsgui" -o qzone-history-gui.exe ./cmd/main.go

# Avec console (pour le débogage)
go build -o qzone-history.exe ./cmd/main.go
```

## Notes techniques

Cet outil **n’est pas** une API officielle de la plateforme ouverte Tencent. Il simule l’accès navigateur aux interfaces internes de QQ Zone (comme ouvrir votre espace dans un navigateur). Les requêtes utilisent des cookies de session et des en-têtes type navigateur, avec limitation du débit pendant l’extraction.

- Les données restent **uniquement sur votre machine** ; aucun envoi vers des serveurs tiers
- Un balayage profond (grand Offset) génère de nombreuses requêtes—réglez les paramètres avec prudence et à vos risques

## Licence

Ce projet est sous [Apache License 2.0](LICENSE).

## Avertissement

Cet outil est destiné à l’apprentissage et à la sauvegarde personnelle de données. Ne l’utilisez pas pour accéder sans autorisation à l’espace d’autrui, pour du scraping commercial ou toute activité illégale. Vous êtes seul responsable des conséquences de son utilisation.
