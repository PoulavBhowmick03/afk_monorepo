export const FACTORY_ADDRESS =
  "0x01a46467a9246f45c8c340f1f155266a26a71c07bd55d36e8d1c7d0d438a2dbc";
export const STARTING_BLOCK = 140_000;
export const STARTING_BLOCK_UNRUG = 615556;
export const LAUNCHPAD_ADDRESS = {
  // SEPOLIA: "0x595d9c14d5b52bae1bd5a88f3aefb521eca956fde4de95e400197f1080fa862",
  // SEPOLIA:"0x5ea5ae49d6449896e096c567350ac639604426c61b16671c37fd9b196ec7fa9",
  // SEPOLIA:"0x732f0364e664ab984049552dcd0aedf7c591ade35d5f3b82e141dd1c987af07"
  SEPOLIA:Deno.env.get("LAUNCHPAD_ADDRESS") ??"0x4fd0893672b60a123606ec9e492e54e795a13e969c6c2600195dcea956ada5d"
};
export {
  constants
} from "https://esm.sh/v135/starknet@5.24.3/dist/index.js";
export const UNRUGGABLE_FACTORY_ADDRESS = "0x01a46467a9246f45c8c340f1f155266a26a71c07bd55d36e8d1c7d0d438a2dbc"
export const NAMESERVICE_ADDRESS = {
  // SEPOLIA: "0x595d9c14d5b52bae1bd5a88f3aefb521eca956fde4de95e400197f1080fa862",
  // SEPOLIA:"0x4fe0ee38c814e0599a5140c5673a233d227ce0be9e22c3acdbee15ac9aefc10"
  SEPOLIA: "0x15dcd3c28c07846fa98d3a40d29446de21b5e6cd8d49a43773da0f237d5ea7f"
  
};
export const SEPOLIA_STREAM_URL="https://sepolia.starknet.a5a.ch"
// export const STREAM_URL=Deno.env.get("ENV_STREAM_URL") ?? "https://sepolia.starknet.a5a.ch"
export const STREAM_URL=Deno.env.get("ENV_STREAM_URL") ?? "https://sepolia.starknet.a5a.ch"