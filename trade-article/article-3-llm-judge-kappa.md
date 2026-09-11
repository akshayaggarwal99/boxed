# Article 3: Our two LLM judges agreed 0.71 of the time. A regex was doing the agreeing.

**Source:** AMP judge paper, Sections 1, 4, 5 and 6. The manuscript and its analysis code are not yet public. The memory system it benchmarks is: https://github.com/akshayaggarwal99/amp and https://pypi.org/project/amp-memory/
**Length:** 1,687 words.
**Audience:** anyone building or reviewing an LLM-as-judge evaluation pipeline. ML engineers, eval leads, benchmark authors.
**Status:** review draft. Every number is verified against the paper's tables. Prose to be retyped by the author before submission.

## Title options

1. Our two LLM judges agreed 0.71 of the time. A regex was doing the agreeing.
2. The Cohen's kappa that validated our judges was measuring our scoring rule
3. How one line of preprocessing turned kappa 0.06 into 0.71 without touching a judge
4. Two judges, one regex, and the agreement statistic that lied to us

**Byline:** Akshay Kumar, Independent Researcher, akumar8@mt.iitr.ac.in

---

If you have shipped an LLM-as-judge pipeline, you have probably shown someone a two-by-two table and a Cohen's kappa. Two judges from different model families, a chance-corrected agreement in the "substantial" band, and the pipeline is declared trustworthy. I did exactly that. The number was 0.714. It was also almost entirely produced by a nine-pattern prefix match that ran before either judge saw a row.

This is the account of how that happened, how big the effect is, and the three lines of code that would have caught it.

## The setup

I was benchmarking agent memory systems on LoCoMo, a set of long multi-session dialogues with question-answer annotations. Four systems, 150 stratified questions each, 600 predictions in all. A fixed generator, `gemini-2.5-flash` at temperature 0.2, so the memory system was the only thing that varied between columns. Two judges, `qwen3:8b` and `deepseek-r1:8b`, run locally through Ollama at temperature 0, neither sharing a family with the generator. That last part is the standard precaution against a model grading its own relatives kindly, and I followed it.

Two of the four systems are mine. I say that now because it matters for how the story reads later.

Early in the work I noticed something. On questions that have a gold answer in the dataset, when a system replied "I don't know" or "I cannot determine that from the conversation," the judges kept marking it correct. So I measured it. Across the 139 predictions that begin with a refusal phrase on an answerable question, `qwen3:8b` scores the refusal as correct 84.2 percent of the time. `deepseek-r1:8b`: 97.1 percent.

Under raw verdicts, refusing costs a system almost nothing. And the systems that refuse most are the ones that retrieve worst, so they collect the most unearned credit. Under raw scoring the four systems sit within 5.3 points of each other. Score refusals as wrong and they spread across 21.3 points. The leniency was hiding the ranking.

So I wrote a rule. Nine refusal prefixes. If a prediction starts with one, write WRONG into both judge columns and do not invoke either judge on that row. It fires on 139 of 600 rows, 23.2 percent.

> **[Figure 3a: `figures/fig-3a-six-hundred-row-split@2x.png`]**
> *Where the reported kappa came from. On 139 rows the rule writes WRONG into both judge columns and the judges never run, so agreement there is 100 percent by construction. On the other 461 rows both judges score, and kappa is 0.064 before and after. The reported 0.056 to 0.714 over all 600 rows is the top branch agreeing with itself.*

Then I recomputed inter-judge agreement. Kappa went from 0.056 to 0.714. Slight to substantial on the usual thresholds. I wrote it up as a reliability gain and recommended the rule as a default.

## Where the gain actually came from

The rule writes both columns on the rows where it fires. So on those 139 rows the judges agree by construction, because I made them agree. That much is obvious once said aloud, and it is not the interesting part.

The interesting part is the other 461 rows. The rule never touches them. Both judges score them. And on those rows kappa is 0.064 before the rule and 0.064 after, identical to four decimal places. Not because anything moved. Because nothing moved. The equality is an identity, not a measurement.

| Sample | N | Kappa before | Kappa after |
|---|---|---|---|
| All 600 rows | 600 | 0.056 | 0.714 |
| Rows the rule writes | 139 | 0.030 | undefined, both raters constant |
| Rows the rule never touches | 461 | 0.064 | 0.064 |

> **[Figure 3b: `figures/fig-3b-kappa-dumbbell@2x.png`]**
> *The kappa gain lived entirely on the rows the rule wrote. Cohen's kappa between qwen3:8b and deepseek-r1:8b before and after the refusal rule: 0.056 to 0.714 over all 600 rows; 0.064 to 0.064, unchanged to four decimals, over the 461 rows the rule never touched.*

The entire gain of 0.658 lives on the rows the rule supplied. On the rows where the judges were actually judging, they agreed about as well as two people flipping coins.

I had reported the top row. I had never split it.

## Two channels, and the bigger one is invisible

Forcing 139 rows to agree is not enough to explain the jump. If you hold the judges' marginal rates where they were and only let observed agreement rise, kappa goes to 0.289. That is the forced-agreement channel, and it is the one anybody would think of.

The rest is prevalence. `deepseek-r1:8b` returned CORRECT on 98.8 percent of all rows. When one rater says yes to nearly everything, the two-by-two table piles up in one cell, expected agreement approaches observed agreement, and kappa collapses toward zero no matter how the raters behave. This is the kappa paradox that Feinstein and Cicchetti described in 1990. My starting value of 0.056 was already an artifact of it.

Now move 139 rows into the WRONG-WRONG cell. The marginals rebalance. Holding observed agreement fixed and letting only the marginals move takes kappa to 0.620. That channel alone is roughly twice the forced-agreement channel, and it is the one nobody thinks of, because it looks like the judges got better when in fact the table got less lopsided.

I can predict the result from the pre-rule table and the count of written rows alone, on the assumption that the written rows are a representative sample. The closed form gives 0.708. The observed value was 0.714.

## It is not about refusals

To make sure I was looking at a property of deterministic overrides and not a quirk of refusal text, I wrote three more filters. Each looks only at the prediction string, never at the gold answer, never at a judge.

| Filter | Rows written | Kappa after | Forced channel alone | Marginal channel alone |
|---|---|---|---|---|
| Refusal prefix | 139 | 0.714 | 0.289 | 0.620 |
| No final punctuation | 136 | 0.822 | 0.591 | 0.589 |
| Under 30 characters | 165 | 0.737 | 0.299 | 0.645 |
| Unresolved relative time ("last week") | 116 | 0.636 | 0.134 | 0.603 |

All four start from 0.056. The truncation filter writes not a single refusal row and inflates kappa more than my rule did. The deixis filter shares no rows with the refusal set either, and its marginal channel is more than four times its forced channel. Any deterministic step upstream of the judges opens both channels. The size of the gain is governed by how skewed the sample already was, which is to say, by exactly the condition that makes a practitioner reach for a filter in the first place.

Raking the pre-rule table to balanced marginals, which preserves the judges' odds ratio, puts them at kappa 0.454. Moderate agreement. Not slight, not substantial. That is probably the honest number for this judge pair, and no version of my pipeline ever reported it.

## The second judge was a constant

The marginals point at one more thing. If a judge says CORRECT 98.8 percent of the time, how much is it contributing?

I replaced `deepseek-r1:8b` with a rater that returns CORRECT unconditionally and recomputed raw agreement with `qwen3:8b`. It moved from 83.83 percent to 83.67 percent. A difference of 0.17 percentage points. Kappa cannot register the substitution at all, because a constant rater has kappa zero by definition.

On the 461 untouched rows there is not one case where `qwen3:8b` said CORRECT and `deepseek-r1:8b` said WRONG. All 73 disagreements run the other way. The second judge never exercised scrutiny in the direction that would have shown it was independent.

I chose it for family diversity, which was the right instinct. What I got was one judge stricter than the other, and a statistic that could not tell me so. At 8 billion parameters with thinking disabled, on this task, it behaves near-constantly. That is not a verdict on the model. It is a verdict on my not having checked.

## What I retracted, and what I kept

Two claims from the earlier version are withdrawn: that the rule improved the reliability of the judge pair, and that after the rule the judges contributed independent evidence on the non-refusal subset. Neither is true.

The measurement that led me to write the rule holds. On answerable questions, both judges credit a refusal as correct most of the time, and a system that retrieves badly and refuses often will look better than it is. Taking that credit away was right. Taking it away by overwriting both judge columns is what destroyed the statistic that would have told me whether I could trust the judges at all.

Scoring a refusal as wrong is also a modelling choice I should state plainly. On a false-premise question, a refusal is arguably the right answer. LoCoMo has an adversarial category for that case, and I have two such questions per system, too few to say anything.

## Three lines of code

None of what follows is new. It is ordinary inter-rater hygiene. The claim is only that a pipeline with any deterministic pre-judgment step needs it and usually does not get it.

**Report kappa on the rows the protocol does not determine.** If an override writes some rows, the denominator is the rest. For me that is 461 rows and 0.064, not 600 rows and 0.714.

**Report each judge's marginal rate of the positive class.** Mine were 83.7 percent and 98.8 percent. Anyone reading the second number would have asked the question I did not.

**Run a constant-rater control.** Replace each judge in turn with a rater that always returns the majority class and recompute agreement with the other. If the number barely moves, you have one judge, not two.

When the restricted number is bad, as mine is, there are three honest responses. Treat the pair as one rater and report that judge with its marginal. Replace the near-constant judge. Or report raw agreement on the undetermined rows with both marginals and make no reliability claim. What is not available is reporting full-sample kappa and calling the pipeline validated, which is what I did.

I am not proposing a replacement coefficient. Prevalence-adjusted indices exist. The problem here is not the formula. It is the denominator. And note that excluding the affected rows instead of recoding them also moves the marginals of what remains, without the visibility of a cell sitting at 100 percent to tip you off.

The memory system under test, the benchmark harness and the judge code are open source at github.com/akshayaggarwal99/amp; the per-question verdicts and the decomposition script will follow with the paper. If you have a judge pipeline with a filter in front of it, the split in the first table takes about ten minutes to reproduce on your own data, and I would be curious what you find.

AMP repository: https://github.com/akshayaggarwal99/amp
Package: https://pypi.org/project/amp-memory/
