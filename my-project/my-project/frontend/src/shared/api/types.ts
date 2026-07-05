// Contract-shaped types (specs/001-fluentdev-pwa/contracts/openapi.yaml).
// Kept in sync with the generated schema (npm run generate:api).

export type ProficiencyLevel = 'basic' | 'intermediate' | 'advanced';

export interface User {
  id: string;
  email: string;
  displayName: string;
  proficiencyLevel: ProficiencyLevel | null;
  currentStreak: number;
  longestStreak: number;
}

export interface Problem {
  type: string;
  title: string;
  status: number;
  detail?: string;
}

export type QuestionType = 'choice' | 'listening_choice' | 'order';

export interface NextQuestion {
  id: string;
  questionType: QuestionType;
  prompt: string;
  options: string[];
  audioAssetUrl: string | null;
}

export interface PlacementState {
  status: 'active' | 'completed';
  questionsServed: number;
  currentBand?: 'A1' | 'A2' | 'B1' | 'B2' | 'C1';
  nextQuestion: NextQuestion | null;
  assignedLevel: ProficiencyLevel | null;
}

export type ExerciseType =
  | 'translate'
  | 'fill_blank'
  | 'listening_choice'
  | 'listening_order'
  | 'speaking';

export interface Exercise {
  id: string;
  exerciseType: ExerciseType;
  promptContext: string;
  options: string[] | null;
  audioAssetUrl: string | null;
  targetSentence: string | null;
}

export interface LessonSummary {
  id: string;
  title: string;
  xpReward: number;
  completed: boolean;
}

export interface Module {
  id: string;
  title: string;
  description: string;
  themeType: 'travel' | 'tech';
  difficultyLevel: ProficiencyLevel;
  locked: boolean;
  lessons: LessonSummary[];
}

export interface Lesson {
  id: string;
  title: string;
  pedagogicalFocus: string;
  xpReward: number;
  exercises: Exercise[];
}

export interface AttemptResult {
  correct: boolean;
  accuracyScore: number;
  toleratedTypos: string[];
  expectedAnswer: string | null;
  lessonCompleted: boolean;
  xpAwarded: number;
}

export interface SpeechResult {
  similarity: number;
  passed: boolean;
  transcript: string;
  missedWords: string[];
  xpAwarded: number;
}

export interface HeatmapDay {
  date: string;
  interactions: number;
  level: 0 | 1 | 2 | 3 | 4;
}

export interface Dashboard {
  currentStreak: number;
  longestStreak: number;
  totalXp: number;
  heatmap: HeatmapDay[];
  dueReviews: number;
}

export interface ReviewItem {
  id: string;
  exercise: Exercise;
  dueAt: string;
  failureCount: number;
}
