"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { apiClient, ApiError } from "@/lib/api-client";
import { Button } from "@/components/ui/Button";
import { Label } from "@/components/ui/Label";

export default function LoginPage() {
  const router = useRouter();
  const [step, setStep] = useState<"phone" | "otp">("phone");
  const [phone, setPhone] = useState("");
  const [otp, setOtp] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const displayPhone = phone ? `+91 ${phone}` : "";

  async function handleSendOTP(e: React.FormEvent) {
    e.preventDefault();
    const cleaned = phone.replace(/\s/g, "");
    if (cleaned.length !== 10) {
      setError("Please enter a valid 10-digit phone number");
      return;
    }
    setError("");
    setLoading(true);
    try {
      await apiClient.post("/api/auth/login", { phone: cleaned });
      setStep("otp");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to send OTP");
    } finally {
      setLoading(false);
    }
  }

  async function handleVerifyOTP(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const cleaned = phone.replace(/\s/g, "");
      await apiClient.post("/api/auth/verify-otp", { phone: cleaned, otp });
      router.push("/");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Invalid OTP");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <h2 className="text-lg font-semibold text-gray-900 mb-1">Welcome back</h2>
      <p className="text-sm text-gray-500 mb-6">Sign in to your LandIntel account</p>

      {step === "phone" ? (
        <form onSubmit={handleSendOTP} className="space-y-4">
          <div>
            <Label htmlFor="phone" required>Phone Number</Label>
            <div className="mt-1 flex rounded-lg border border-gray-300 focus-within:ring-2 focus-within:ring-emerald-500 focus-within:border-emerald-500 overflow-hidden">
              <span className="inline-flex items-center px-3 bg-gray-50 text-gray-500 text-sm border-r border-gray-300 select-none">
                +91
              </span>
              <input
                id="phone"
                type="tel"
                inputMode="numeric"
                placeholder="98765 43210"
                value={phone}
                onChange={(e) => setPhone(e.target.value.replace(/[^0-9\s]/g, "").slice(0, 12))}
                maxLength={12}
                required
                className="flex-1 px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:outline-none"
              />
            </div>
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <Button type="submit" loading={loading} className="w-full">
            Send OTP
          </Button>
        </form>
      ) : (
        <form onSubmit={handleVerifyOTP} className="space-y-4">
          <div className="bg-emerald-50 rounded-lg p-3 text-sm text-emerald-800">
            OTP sent to <span className="font-semibold">{displayPhone}</span>
          </div>
          <div>
            <Label htmlFor="otp" required>Enter OTP</Label>
            <input
              id="otp"
              type="text"
              inputMode="numeric"
              placeholder="Enter 6-digit OTP"
              value={otp}
              onChange={(e) => setOtp(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))}
              maxLength={6}
              required
              autoFocus
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 tracking-widest text-center text-lg font-mono focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 focus:outline-none"
            />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <Button type="submit" loading={loading} className="w-full">
            Verify OTP
          </Button>
          <button
            type="button"
            onClick={() => { setStep("phone"); setOtp(""); setError(""); }}
            className="text-sm text-emerald-600 hover:text-emerald-500"
          >
            Change phone number
          </button>
        </form>
      )}

      <p className="mt-6 text-center text-sm text-gray-500">
        Don&apos;t have an account?{" "}
        <Link href="/register" className="text-emerald-600 hover:text-emerald-500 font-medium">
          Register
        </Link>
      </p>
    </div>
  );
}
