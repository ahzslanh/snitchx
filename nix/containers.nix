# oci container definitions for snitchx
# builds containers based on different base images: alpine, debian trixie, ubuntu
#
# base images are pinned by imageDigest (immutable content hash), not by tag.
# even if the upstream tag gets a new image, builds remain reproducible.
#
# to update base image hashes, run:
#   nix-prefetch-docker --image-name alpine --image-tag 3.21
#   nix-prefetch-docker --image-name debian --image-tag trixie-slim
#   nix-prefetch-docker --image-name ubuntu --image-tag 24.04
#
# this outputs both imageDigest and sha256 values needed below
{ pkgs, snitchx }:
let
  commonConfig = {
    name = "snitchx";
    tag = snitchx.version;
    config = {
      Entrypoint = [ "${snitchx}/bin/snitchx" ];
      Env = [ "PATH=/bin" ];
      Labels = {
        "org.opencontainers.image.title" = "snitchx";
        "org.opencontainers.image.description" = "cross-platform security inspection: network, processes, services, compliance";
        "org.opencontainers.image.source" = "https://github.com/ahzslanh/snitchx";
        "org.opencontainers.image.licenses" = "MIT";
      };
    };
  };

  # alpine-based container
  alpine = pkgs.dockerTools.pullImage {
    imageName = "alpine";
    imageDigest = "sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c";
    sha256 = "sha256-WNbRh44zld3lZtKARhdeWFte9JKgD2bgCuKzETWgGr8=";
    finalImageName = "alpine";
    finalImageTag = "3.21";
  };

  # debian trixie (testing) based container
  debianTrixie = pkgs.dockerTools.pullImage {
    imageName = "debian";
    imageDigest = "sha256:e711a7b30ec1261130d0a121050b4ed81d7fb28aeabcf4ea0c7876d4e9f5aca2";
    sha256 = "sha256-W/9A7aaPXFCmmg+NTSrFYL+QylsAgfnvkLldyI18tqU=";
    finalImageName = "debian";
    finalImageTag = "trixie-slim";
  };

  # ubuntu based container
  ubuntu = pkgs.dockerTools.pullImage {
    imageName = "ubuntu";
    imageDigest = "sha256:c35e29c9450151419d9448b0fd75374fec4fff364a27f176fb458d472dfc9e54";
    sha256 = "sha256-0j8xM+mECrBBHv7ZqofiRaeSoOXFBtLYjgnKivQztS0=";
    finalImageName = "ubuntu";
    finalImageTag = "24.04";
  };

  # scratch container (minimal, just the snitchx binary)
  scratch = pkgs.dockerTools.buildImage {
    name = "snitchx";
    tag = "${snitchx.version}-scratch";
    copyToRoot = pkgs.buildEnv {
      name = "snitchx-root";
      paths = [ snitchx ];
      pathsToLink = [ "/bin" ];
    };
    config = commonConfig.config;
  };

in
{
  snitchx-alpine = pkgs.dockerTools.buildImage {
    name = "snitchx";
    tag = "${snitchx.version}-alpine";
    fromImage = alpine;
    copyToRoot = pkgs.buildEnv {
      name = "snitchx-root";
      paths = [ snitchx ];
      pathsToLink = [ "/bin" ];
    };
    config = commonConfig.config;
  };

  snitchx-debian = pkgs.dockerTools.buildImage {
    name = "snitchx";
    tag = "${snitchx.version}-debian";
    fromImage = debianTrixie;
    copyToRoot = pkgs.buildEnv {
      name = "snitchx-root";
      paths = [ snitchx ];
      pathsToLink = [ "/bin" ];
    };
    config = commonConfig.config;
  };

  snitchx-ubuntu = pkgs.dockerTools.buildImage {
    name = "snitchx";
    tag = "${snitchx.version}-ubuntu";
    fromImage = ubuntu;
    copyToRoot = pkgs.buildEnv {
      name = "snitchx-root";
      paths = [ snitchx ];
      pathsToLink = [ "/bin" ];
    };
    config = commonConfig.config;
  };

  snitchx-scratch = scratch;

  oci-default = pkgs.dockerTools.buildImage {
    name = "snitchx";
    tag = snitchx.version;
    fromImage = alpine;
    copyToRoot = pkgs.buildEnv {
      name = "snitchx-root";
      paths = [ snitchx ];
      pathsToLink = [ "/bin" ];
    };
    config = commonConfig.config;
  };
}

