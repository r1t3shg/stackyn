import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error:', error, errorInfo);
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-[var(--app-bg)] flex items-center justify-center p-4">
          <div className="max-w-md w-full bg-[var(--surface)] rounded-lg border border-[var(--border-subtle)] p-6">
            <h1 className="text-2xl font-bold text-[var(--error)] mb-4">Something went wrong</h1>
            <p className="text-[var(--text-primary)] mb-4">
              {this.state.error?.message || 'An unexpected error occurred'}
            </p>
            <button
              onClick={() => {
                this.setState({ hasError: false, error: null });
                window.location.reload();
              }}
              className="bg-[var(--primary)] hover:bg-[var(--primary-hover)] text-[var(--app-bg)] font-medium py-2 px-4 rounded-lg transition-colors"
            >
              Reload Page
            </button>
            <details className="mt-4">
              <summary className="cursor-pointer text-sm text-[var(--text-secondary)]">Error Details</summary>
              <pre className="mt-2 text-xs bg-[var(--elevated)] text-[var(--text-primary)] p-2 rounded overflow-auto max-h-40 border border-[var(--border-subtle)]">
                {this.state.error?.stack}
              </pre>
            </details>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;

