import { adminAPI } from "./Admin";
import { authAPI } from "./Auth";
import { catalogAPI } from "./Catalog";
import { favoritesAPI } from "./Favorites";
import { historyAPI } from "./History";
import { liveAPI } from "./Live";
import { pushAPI } from "./Push";
import { serviceAlertAPI } from "./ServiceAlert";
import { settingsAPI } from "./Settings";
import { socialAPI } from "./Social";
import { sportsAPI } from "./Sports";
import { streamAPI } from "./Stream";
import { versionAPI } from "./Version";

export { ApiError } from "./Request";

export default class Net {

  static readonly Admin = adminAPI;
  static readonly Auth = authAPI;
  static readonly Catalog = catalogAPI;
  static readonly Favorites = favoritesAPI;
  static readonly History = historyAPI;
  static readonly Live = liveAPI;
  static readonly Push = pushAPI;
  static readonly ServiceAlert = serviceAlertAPI;
  static readonly Settings = settingsAPI;
  static readonly Social = socialAPI;
  static readonly Sports = sportsAPI;
  static readonly Stream = streamAPI;
  static readonly Version = versionAPI;

}
