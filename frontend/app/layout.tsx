import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Trip OS",
  description: "The group travel operating system, built for how India actually plans trips.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
