import { api } from "@/api/client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

import { Component } from "react";
import { motion } from "framer-motion";

interface AuthPageProps {

  onSuccess: () => void;

}

interface AuthPageState {

  mode: "login" | "register";

  email: string;
  password: string;
  accessCode: string;

  error: string;

  loading: boolean;

}

export class AuthPage extends Component<AuthPageProps, AuthPageState> {

  state: AuthPageState = {

    mode: "login",

    email: "",
    password: "",
    accessCode: "",

    error: "",

    loading: false,

  };

  submit = async (e: React.FormEvent) => {

    e.preventDefault();

    const { mode, email, password, accessCode } = this.state;

    this.setState({ loading: true, error: "" });

    try {

      if (mode === "login") {

        await api.login(email, password);

      } else {

        await api.register(email, password, accessCode);

      }

      this.props.onSuccess();

    } catch (err) {

      this.setState({

        error: err instanceof Error ? err.message : "authentication failed",
        loading: false,

      });

    }

  };

  render() {

    const { mode, email, password, accessCode, error, loading } = this.state;

    return (

      <div className="flex min-h-screen items-center justify-center px-4">

        <motion.div className="w-full max-w-sm"

          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}

          transition={{ duration: 0.5 }}

        >

          <div className="mb-8 text-center">

            <div className="mb-2">

              <span className="text-lg font-semibold tracking-tight">

                Streamly <span className="font-light text-foreground-muted">Web</span>

              </span>

            </div>

            <p className="text-sm text-foreground-muted">

              {mode === "login" ? "Sign in to continue" : "Create your account"}

            </p>

          </div>

          <form onSubmit={this.submit} className="space-y-4">

            <div>

              <label className="mb-1.5 block text-xs text-foreground-muted">Email</label>

              <Input type="email"

                value={email}
                onChange={(e) => this.setState({ email: e.target.value })}

                required
                autoComplete="email"

              />

            </div>

            <div>

              <label className="mb-1.5 block text-xs text-foreground-muted">Password</label>

              <Input type="password"

                value={password}
                onChange={(e) => this.setState({ password: e.target.value })}

                required
                minLength={8}
                autoComplete={mode === "login" ? "current-password" : "new-password"}

              />

            </div>

            {mode === "register" && (

              <div>

                <label className="mb-1.5 block text-xs text-foreground-muted">Access Code</label>

                <Input value={accessCode}

                  onChange={(e) => this.setState({ accessCode: e.target.value })}

                  required
                  placeholder="Required for registration"

                />

              </div>

            )}

            {error && <p className="text-xs text-red-400">{error}</p>}

            <Button type="submit" className="w-full" disabled={loading}>

              {loading ? "Please wait..." : mode === "login" ? "Sign In" : "Create Account"}

            </Button>

          </form>

          <p className="mt-6 text-center text-xs text-foreground-muted">

            {mode === "login" ? "Need an account?" : "Already have an account?"}{" "}

            <button type="button"

              onClick={() =>

                this.setState({ mode: mode === "login" ? "register" : "login", error: "" })

              }

              className="text-foreground underline-offset-2 hover:underline"

            >

              {mode === "login" ? "Register" : "Sign in"}

            </button>

          </p>

        </motion.div>

      </div>

    );

  }

}
