# Trade article packet for the review team

Three long-form drafts, 8 Sep 2026. Venue to be decided after review; `venues.md`
has the ranked list with word caps and contacts.

| # | File | Words | Source | One line |
|---|---|---|---|---|
| 1 | `article-1-hardening-yama-scoring.md` | 1,626 | Boxed paper | Hardening made containers faster; a Kata microVM exposed a protection that belonged to the host kernel; grep-based escape tests misscored 2 of 12 vectors |
| 2 | `article-2-where-the-time-goes.md` | 1,744 | Boxed paper | The model is 85 percent of an agent step; 4x cores bought 1.2x sandboxes because the daemon is the bottleneck; two hardening flags tripled a microVM's boot |
| 3 | `article-3-llm-judge-kappa.md` | 1,687 | AMP judge paper | A refusal regex moved Cohen's kappa from 0.06 to 0.71 without touching a judge; on the 461 rows it never wrote, kappa stayed 0.064 |

Also here: `draft-venturebeat.md`, the 1,031-word cut of article 1 for VentureBeat's
800-1200 cap, and `pitch.md`, a cover note for outlets that take pitches.

## How the three relate

Articles 1 and 2 draw on the same measurement campaign but share no findings.
Article 1 is the security and measurement-method story. Article 2 is the
capacity-planning story. Either can run without the other, and the two should
not go to the same outlet.

Article 3 is from a different paper and a different field (LLM evaluation).
Its audience is ML and eval engineers rather than infrastructure. It is the most
broadly applicable of the three: anyone with a filter in front of an LLM judge
can reproduce the split in ten minutes on their own data.

## What is deliberately not here

The OpenHands comparison (7.6 s per lifecycle, 9 of 12 vectors stopped, cloud
metadata endpoint and service-account token reachable from the sandbox). It is
the highest-traffic finding in the Boxed paper and it stays out of every article
until the OpenHands maintainers have been notified. Article 4, after disclosure.

## Before any of these is submitted

1. **The author retypes the prose.** Every number is verified against the raw
   traces or the paper's tables and the structure is settled, but the sentences
   are a draft. VentureBeat and No Jitter ban AI-assisted writing outright, and
   the paper standard in `research-paper/references/human-prose.md` asks for the
   same. Read each aloud; where you would have said it differently, say it
   differently.
2. **Pick one title per article.** Options are at the top of each file.
3. **Confirm the byline.** All three carry Akshay Kumar, Independent Researcher,
   akumar8@mt.iitr.ac.in. Article 3 also discloses that two of the four
   benchmarked systems are the author's own; keep that line.
4. **One outlet at a time.** Everything on the venue list wants exclusivity.

## Regenerating the numbers

Articles 1 and 2: every figure expands from `paper-v2/tables/numbers*.tex`,
which `make tables` in `paper-v2/` regenerates from `bench/results/`.
Article 3: from the tables in `sementic-context-protocol/amp/paper-judge/sections/05_results.tex`.
