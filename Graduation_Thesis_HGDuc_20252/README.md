# Graduation Thesis

My graduation thesis, written in LaTeX. No local TeX installation is needed — everything runs inside Docker.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

## Build

```bash
# One-shot build → produces build/main.pdf
docker compose run --rm build
```

## Useful commands

| Command | What it does |
| --- | --- |
| `docker compose run --rm build` | Build the PDF once |
| `docker compose run --rm watch` | Rebuild automatically on every file change |
| `docker compose run --rm clean` | Remove all build artefacts |
| `docker compose run --rm shell` | Open an interactive shell inside the container |

## Project structure

```
.
├── main.tex          # Entry point
├── paper.tex         # Paper version
├── reference.bib     # Bibliography
├── glossary.tex      # Glossary / acronyms
├── Chapter/          # Individual chapters
├── Figure/           # Figures and images
├── Cover.tex         # Thesis cover page
├── Cover2.tex        # Alternative cover page
├── docker-compose.yml
└── build/            # Build output (main.pdf ends up here)
```
