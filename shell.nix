(import
  (fetchTarball {
    url = "https://github.com/edolstra/flake-compat/archive/12c64ca55c0114d1f539ca0ab4ffa7ca886fc590.tar.gz";
    sha256 = "0jm6nzb83wa6ai17ly9fzpqc40wg1viib8kyqym4qabdk8j9cryx";
  })
  { src = ./.; }
).shellNix
