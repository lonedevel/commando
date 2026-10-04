class Commando < Formula
  desc "Colorful TUI that turns a command's man page into a form for its options"
  homepage "https://github.com/lonedevel/commando"
  url "https://github.com/lonedevel/commando/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "42a718d694cb92f8f59a5e398e1287ab51447fff4107cc3e6e69d17bd642db43"
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
