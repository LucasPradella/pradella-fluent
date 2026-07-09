// Route table with the auth/placement gate:
// new user → placement; placed user → dashboard; anonymous → login.
import { Navigate, Route, Routes } from 'react-router-dom';
import { useMe } from '../shared/api/hooks';
import { Shell } from './shell';
import { LoginPage } from '../features/onboarding/login';
import { RegisterPage } from '../features/onboarding/register';
import { PlacementFlow } from '../features/placement/placement-flow';
import { TracksPage } from '../features/lessons/tracks';
import { LessonPlayer } from '../features/lessons/lesson-player';
import { DashboardPage } from '../features/dashboard/dashboard';
import { ReviewSession } from '../features/review/review-session';

function Gate({ children }: { children: React.ReactNode }) {
  const { data: me, isLoading } = useMe();
  if (isLoading) return <p style={{ padding: 24 }}>Carregando…</p>;
  if (!me) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function Home() {
  const { data: me, isLoading } = useMe();
  if (isLoading) return <p style={{ padding: 24 }}>Carregando…</p>;
  if (!me) return <Navigate to="/login" replace />;
  if (!me.proficiencyLevel) return <Navigate to="/placement" replace />;
  return <Navigate to="/dashboard" replace />;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/placement"
        element={
          <Gate>
            <PlacementFlow />
          </Gate>
        }
      />
      <Route
        element={
          <Gate>
            <Shell />
          </Gate>
        }
      >
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/tracks" element={<TracksPage />} />
        <Route path="/lessons/:lessonId" element={<LessonPlayer />} />
        <Route path="/review" element={<ReviewSession />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
