# Feature Specification: FluentDev — English Learning PWA for Tech Professionals

**Feature Branch**: `001-fluentdev-pwa`

**Created**: 2026-07-04

**Status**: Draft

**Input**: User description: "prd.md — Plataforma PWA de Ensino de Inglês para Profissionais de Tecnologia (FluentDev). PWA educacional para falantes de pt-BR (perfis: desenvolvedor e viajante) com teste de nivelamento adaptativo, lições baseadas em tarefas, avaliação de fala por transcrição, repetição espaçada, gamificação estilo GitHub (streak + heatmap), tema escuro acessível e suporte offline."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Onboarding and Adaptive Placement Test (Priority: P1)

A new user signs up with minimal friction (social sign-in or e-mail) and is immediately guided into a short adaptive placement test. The test starts at medium difficulty and adjusts up or down based on performance, ending after a bounded number of interactions with a proficiency diagnosis (CEFR-aligned: Basic A1–A2, Intermediate B1–B2, or Advanced C1). Based on the result, the user is placed on a learning track and sees which content is unlocked for them.

**Why this priority**: It is the entry point for every user and produces the proficiency level that all other features (lessons, exercises, spaced repetition) depend on. On its own it already delivers value: a fast, accurate English diagnosis for the target audience.

**Independent Test**: Can be fully tested by creating an account, completing the placement test, and verifying that a proficiency level is assigned and the matching learning track is unlocked — without any lesson content beyond the calibrated question bank.

**Acceptance Scenarios**:

1. **Given** a visitor on the landing screen, **When** they choose a social sign-in provider (GitHub or Google) or register with e-mail and password, **Then** an account is created and the placement test starts immediately without additional forms.
2. **Given** a user answering a testlet (group of 3 questions) with more than 70% accuracy, **When** the next testlet is served, **Then** its questions are drawn from a higher difficulty band.
3. **Given** a user answering a testlet with less than 40% accuracy, **When** the next testlet is served, **Then** its questions are drawn from a lower difficulty band.
4. **Given** a user who has completed 12 scored interactions, **When** the 12th answer is submitted, **Then** the test ends and a proficiency level (Basic, Intermediate, or Advanced) is displayed with an explanation of the assigned track.
5. **Given** a user with an assigned level of Basic, **When** they view the learning tracks, **Then** Intermediate and Advanced tracks are visibly locked.
6. **Given** a user who abandons the placement test midway, **When** they return, **Then** they can resume from where they stopped or restart the test.

---

### User Story 2 - Task-Based Lessons (Writing and Listening) (Priority: P2)

A placed learner opens their track and completes lessons organized around realistic tasks — either tech scenarios ("An internal 500 error occurred on the server; explain the situation") or travel scenarios (airport, hotel, car rental). Within a lesson, they complete writing exercises (translate or complete sentences, tolerant of small typos) and listening exercises (play audio, then answer multiple-choice or reorder word blocks to reconstruct the sentence).

**Why this priority**: This is the core learning loop and the bulk of the product's daily value. It can ship before speech evaluation and still be a usable learning product.

**Independent Test**: Can be tested by assigning a learner a track (manually or via US1), completing a full lesson containing writing and listening exercises, and verifying scoring, typo tolerance, and progress recording.

**Acceptance Scenarios**:

1. **Given** a learner on the Basic track, **When** they open their track, **Then** they see modules themed as Travel or Tech, each containing ordered lessons appropriate to their level.
2. **Given** a writing exercise expecting "I deployed the new feature yesterday", **When** the learner submits "I deployed the new featur yesterday" (minor typo), **Then** the answer is accepted as correct with the typo highlighted.
3. **Given** a writing exercise, **When** the learner submits a semantically wrong answer, **Then** the answer is rejected and the expected answer is shown as feedback.
4. **Given** a listening exercise, **When** the learner plays the audio, **Then** they can replay it and answer via multiple choice or by ordering word blocks to form the heard sentence.
5. **Given** a learner who completes all exercises in a lesson, **When** the last exercise is submitted, **Then** the lesson is marked complete, experience points are awarded, and the completion is recorded with a timestamp and accuracy score.

---

### User Story 3 - Speaking Practice with Automatic Feedback (Priority: P3)

In a speaking exercise, the learner sees a sentence in American English, presses the microphone button, and speaks it aloud (up to 30 seconds). The system transcribes the recording, compares it to the target sentence, and shows a similarity percentage. At 80% similarity or above the task passes; omitted or unintelligible words are highlighted in a desaturated contrast color so the learner can retry.

**Why this priority**: Speech evaluation is the platform's core differentiator, but it depends on the lesson engine (US2) as its host and is the highest technical-risk component — so it ships after the base loop is stable.

**Independent Test**: Can be tested by opening any speaking exercise, recording a sentence, and verifying that a transcription-based similarity score and word-level feedback are shown within the latency budget.

**Acceptance Scenarios**:

1. **Given** a speaking exercise, **When** the learner presses the microphone button for the first time, **Then** the app requests microphone permission and clearly explains why it is needed.
2. **Given** an active recording, **When** 30 seconds elapse, **Then** recording stops automatically and processing begins.
3. **Given** a recorded attempt whose transcription is ≥80% similar to the target sentence, **When** the result renders, **Then** the exercise is marked successful and the similarity percentage is displayed.
4. **Given** a recorded attempt with omitted or unintelligible words, **When** the result renders, **Then** those words are highlighted in a desaturated red and the learner may retry immediately.
5. **Given** a learner who denies microphone permission, **When** they reach a speaking exercise, **Then** they are offered instructions to re-enable it and an option to skip the exercise without blocking lesson completion.
6. **Given** a recording submitted on a stable mobile connection, **When** processing completes, **Then** the feedback appears within the latency budget (see SC-004).

---

### User Story 4 - Streak, Activity Heatmap, and Spaced Review (Priority: P4)

The learner's dashboard shows a consecutive-day streak counter and a GitHub-style contribution heatmap covering the last 90 days, where each day's square gets more color-saturated with more completed activity. Items the learner recently failed are automatically queued for spaced review and resurface as quick exercises on subsequent days; completing them counts toward the streak and heatmap.

**Why this priority**: Retention mechanics multiply the value of the learning loop but deliver nothing without it; they layer cleanly on top of US1–US3 activity data.

**Independent Test**: Can be tested by completing activity on multiple (simulated) days and verifying streak increments/resets, heatmap saturation levels, and that failed items reappear in the review queue on later days.

**Acceptance Scenarios**:

1. **Given** a learner who completed at least one lesson or review yesterday and one today, **When** they view the dashboard, **Then** the streak counter shows 2.
2. **Given** a learner with no completed activity yesterday, **When** they complete a lesson today, **Then** the streak counter shows 1 (reset), while the longest-streak record is preserved.
3. **Given** 90 days of varied activity, **When** the dashboard renders, **Then** the heatmap shows one square per day with color saturation proportional to that day's completed interactions.
4. **Given** an exercise the learner failed two days ago, **When** they open the app today, **Then** that item is offered in a quick review session, and completing it is logged as a spaced-repetition review.

---

### User Story 5 - Installable, Offline-Resilient App Experience (Priority: P5)

The learner installs the app to their home screen from the browser on Android or iOS and gets a full-screen standalone experience. On later visits — including with no connectivity — the app shell (navigation, headers, structural panels) loads almost instantly, previously loaded profile and progress data render from local storage, and progress made is synchronized to the cloud so nothing is lost even if the device purges local data.

**Why this priority**: Installability and offline resilience elevate the experience and protect user data, but the product is already usable online-only in a browser tab.

**Independent Test**: Can be tested by installing the app on Android and iOS devices, loading it in airplane mode, verifying the shell renders under the load-time budget with cached data, and confirming progress made near connectivity loss appears in the cloud account afterward.

**Acceptance Scenarios**:

1. **Given** a supported mobile browser, **When** the learner adds the app to the home screen, **Then** it launches full-screen in standalone mode with proper icons.
2. **Given** a returning user with no network connection, **When** they open the installed app, **Then** the app shell renders in under 1.5 seconds with their last-known profile and progress data.
3. **Given** an offline user, **When** they reach a feature that requires connectivity (speaking evaluation, sign-in), **Then** the app explains that this feature needs a connection instead of failing silently.
4. **Given** a user whose device purged locally stored data after 7+ days of inactivity, **When** they sign in again, **Then** their full progress is restored from the cloud with no data loss.

---

### Edge Cases

- **Microphone permission denied or unavailable**: speaking exercises must offer skip-and-continue so a lesson can always be completed; the skipped item enters the review queue.
- **Speech processing unavailable or timing out**: the learner sees a clear retry option; the attempt is never counted as a failure due to infrastructure error.
- **Ambient noise / unintelligible audio**: the system reports that it could not understand the recording and prompts a retry rather than scoring 0% silently.
- **Placement test abandoned midway**: progress within the test is kept so the user can resume or restart; no proficiency level is assigned until completion.
- **Streak across timezones / midnight boundary**: streak days are computed in the user's local timezone; activity at 23:59 and 00:01 counts as two distinct days.
- **Local data purge (7-day eviction on some mobile platforms)**: all progress writes are synchronized to the cloud optimistically; local storage is a cache, never the source of truth.
- **Repeatedly failed review items**: an item failed multiple times reappears at shorter intervals rather than being dropped from the queue.
- **Simultaneous use on two devices**: latest-synchronized progress wins; streak and heatmap reflect the union of completed activity.
- **Audio asset fails to load in a listening exercise**: the learner can skip the exercise and it is rescheduled, rather than blocking the lesson.

## Requirements *(mandatory)*

### Functional Requirements

**Onboarding & Placement**

- **FR-001**: System MUST allow account creation via GitHub sign-in, Google sign-in, or e-mail with password (stored only in encrypted/hashed form).
- **FR-002**: System MUST start the adaptive placement test immediately after account creation, with no intermediate mandatory forms.
- **FR-003**: System MUST maintain a calibrated question bank spanning CEFR levels A1 through C1 for the placement test.
- **FR-004**: The placement engine MUST serve questions in testlets of 3; a testlet score above 70% raises the next testlet's difficulty band, below 40% lowers it.
- **FR-005**: The placement test MUST end after at most 12 scored interactions and assign exactly one proficiency level: Basic (A1–A2), Intermediate (B1–B2), or Advanced (C1).
- **FR-006**: System MUST unlock the learning track matching the assigned level and keep higher tracks locked until the learner progresses to them.

**Lessons & Exercises**

- **FR-007**: System MUST organize content as Modules (themed Travel or Tech, with a difficulty level and order) containing Lessons, which contain Exercises.
- **FR-008**: Every lesson MUST be framed as a realistic task or scenario (task-based teaching), not as isolated grammar drills.
- **FR-009**: System MUST support writing exercises (translation or sentence completion) whose validation tolerates small typos while rejecting semantically wrong answers.
- **FR-010**: System MUST support listening exercises with replayable audio and answers via multiple choice or word-block ordering.
- **FR-011**: System MUST award experience points on lesson completion and record every exercise interaction with timestamp and accuracy score in an immutable activity log.
- **FR-012**: The MVP content library MUST include at least 20 lessons covering both Travel and Tech themes across the available levels.

**Speaking Evaluation**

- **FR-013**: System MUST let the learner record spoken responses of up to 30 seconds per attempt, with recording stopped automatically at the limit.
- **FR-014**: System MUST transcribe the recording and compute a similarity score against the target sentence; ≥80% similarity marks the exercise successful.
- **FR-015**: System MUST highlight omitted or unintelligible words in a desaturated contrast color and allow immediate retry.
- **FR-016**: System MUST degrade gracefully when the microphone is unavailable or permission is denied: explain, allow skip, and never block lesson completion.
- **FR-017**: System MUST route speech processing to on-device processing when the device is capable, and to remote processing otherwise, transparently to the learner.

**Engagement & Retention**

- **FR-018**: System MUST display a consecutive-day streak counter (resetting after a day with zero completed activity) and preserve the longest-streak record.
- **FR-019**: System MUST display an activity heatmap of the last 90 days with per-day color saturation proportional to completed interactions.
- **FR-020**: System MUST automatically schedule recently failed items into a spaced-review queue that resurfaces them as quick exercises on subsequent days.

**Platform & Data**

- **FR-021**: System MUST be installable to the home screen on Android and iOS with a full-screen standalone experience and scalable icons (192px and 512px).
- **FR-022**: System MUST render the app shell and last-known user data when offline, and clearly signal which features require connectivity.
- **FR-023**: System MUST synchronize all progress to the cloud optimistically so that loss of locally stored data (e.g., platform-enforced 7-day eviction) never loses user progress.
- **FR-024**: All user-facing text and interactive elements MUST meet WCAG 2.1 AA contrast (≥4.5:1); the dark theme MUST use attenuated dark backgrounds (e.g., #121212-class tones, never pure black) with light-gray text and desaturated accent colors.
- **FR-025**: The user interface MUST be presented in Brazilian Portuguese, with learning content in American English.

### Key Entities

- **User**: identity and telemetry base — unique id, sign-in provider link, e-mail, display name, assigned proficiency level, current streak, longest streak, creation date.
- **Module**: semantic grouping of content — title, description, theme (Travel or Tech), difficulty level, sequential order.
- **Lesson**: unit teaching block belonging to a Module — title, pedagogical focus, experience-point reward.
- **Exercise**: interactive task belonging to a Lesson — type (translate, speaking, listening, fill-blank), prompt/context, target answer text, optional audio asset reference.
- **Progress Log Entry**: immutable record of one interaction — user, exercise, completion timestamp, accuracy score, whether it was a spaced-repetition review; feeds streak, heatmap, and placement analytics.
- **Placement Session**: a user's placement test in progress or completed — served testlets, per-question results, current difficulty band, final assigned level.
- **Review Queue Item**: a failed item scheduled for spaced review — user, exercise, next due date, failure count.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can go from first visit to an assigned proficiency level in 10 minutes or less.
- **SC-002**: The placement test never exceeds 12 scored interactions, and repeat takers receive the same level classification at least 80% of the time.
- **SC-003**: Learners can complete a full lesson (all exercise types except speaking) end-to-end without assistance on the first attempt in at least 90% of usability sessions.
- **SC-004**: Speaking feedback (recording → evaluation → rendered result) completes within 3.5 seconds for 95% of attempts on a stable 4G-class connection.
- **SC-005**: After a first successful visit, the app shell loads in under 1.5 seconds on return visits, including with no network connection.
- **SC-006**: The app is installable and runs full-screen standalone on current Android and iOS devices, verified on both platforms before launch.
- **SC-007**: 100% of shipped screens pass WCAG 2.1 AA contrast validation (≥4.5:1) in dark theme.
- **SC-008**: At launch, at least 20 lessons are available spanning both Travel and Tech themes.
- **SC-009**: No user loses recorded progress across devices or local-data eviction: progress visible after re-login matches the cloud record in 100% of sync tests.
- **SC-010**: Failed items resurface for review within the scheduled window in 100% of scheduling tests, and completed reviews register on streak and heatmap.

## Assumptions

- The MVP is entirely free to use; payment, subscription tiers, and the B2B/corporate dashboards from the PRD roadmap (Phase 3) are out of scope for this feature.
- Placement test questions are text- and listening-based (multiple choice / ordering); speaking is not evaluated during placement.
- The interface language is Brazilian Portuguese only for the MVP; learning content targets American English.
- Curriculum content (the 20+ lessons and the calibrated placement question bank) is authored and provided as part of this feature's content workstream; the system stores and serves it but does not generate it dynamically in the MVP (LLM-generated exercises are Phase 3 roadmap).
- Push notifications are out of scope for the MVP (PRD Phase 3).
- Leaderboards and badges are desirable gamification extensions but are not MVP-blocking; streak, heatmap, and XP are the MVP retention mechanics.
- Speaking evaluation requires network connectivity in the MVP; on-device processing (FR-017) may initially apply only to capable desktop-class devices, with all other devices using remote processing.
- Standard privacy practices apply: voice recordings are processed for evaluation and not retained longer than needed for that purpose; users are informed before first recording.

