class Chartmux < Formula
  desc "Charts for terminal UI, PNG, SVG, HTML, and JSON"
  homepage "https://github.com/mertdeveci5/chartmux"
  url "https://github.com/mertdeveci5/chartmux/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "3a6edd164b9c72c8dde13801995f4a4c07122b0d5bc3e7f93733b0392adb48e7"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags:), "./cmd/chartmux"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/chartmux --version")
  end
end
