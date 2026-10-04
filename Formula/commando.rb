class Commando < Formula
  desc "Colorful TUI that turns a command's man page into a form for its options"
  homepage "https://github.com/lonedevel/commando"
  url "https://github.com/lonedevel/commando/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "cc8cc926c202cc0e448b5a8f7678b2d57e8af14833cf7020e430375406bb7340"
  license "Apache-2.0"
  head "https://github.com/lonedevel/commando.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/commando"
  end

  def caveats
    <<~EOS
      To edit the command you're typing with Ctrl-X Ctrl-O, add to your shell config:
        zsh:  eval "$(commando --init zsh)"
        bash: eval "$(commando --init bash)"
        fish: commando --init fish | source
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/commando --version")
    assert_match "bindkey", shell_output("#{bin}/commando --init zsh")

    # Parse a manual without needing a terminal.
    ENV["COMMANDO_CACHE_DIR"] = testpath/"cache"
    output = shell_output("#{bin}/commando --dump --no-cache ls")
    assert_match "\"names\"", output
  end
end
