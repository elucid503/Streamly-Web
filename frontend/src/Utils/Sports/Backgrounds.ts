const IMAGES: Record<string, string> = {

  baseball: "https://upload.wikimedia.org/wikipedia/commons/thumb/e/e7/Tommy_Milone_gives_up_a_home_run_to_Mike_Trout_on_May_21%2C_2017.jpg/1280px-Tommy_Milone_gives_up_a_home_run_to_Mike_Trout_on_May_21%2C_2017.jpg",
  basketball: "https://upload.wikimedia.org/wikipedia/commons/thumb/0/06/Steph_Curry_%2851915116957%29.jpg/1280px-Steph_Curry_%2851915116957%29.jpg",
  "american-football": "https://upload.wikimedia.org/wikipedia/commons/d/df/Larry_Fitzgerald_catches_TD_at_2009_Pro_Bowl.jpg",
  football: "https://upload.wikimedia.org/wikipedia/commons/4/42/Football_in_Bloomington%2C_Indiana%2C_1995.jpg",
  soccer: "https://upload.wikimedia.org/wikipedia/commons/4/42/Football_in_Bloomington%2C_Indiana%2C_1995.jpg",
  afl: "https://upload.wikimedia.org/wikipedia/commons/f/f3/Archie_Smith.jpg",
  rugby: "https://upload.wikimedia.org/wikipedia/commons/f/fd/Fraus04rugby13.jpg",
  "motor-sports": "https://upload.wikimedia.org/wikipedia/commons/1/14/2010_Malaysian_GP_opening_lap.jpg",
  fight: "https://upload.wikimedia.org/wikipedia/commons/4/49/UFC_131_Carwin_vs._JDS.jpg",
  mma: "https://upload.wikimedia.org/wikipedia/commons/4/49/UFC_131_Carwin_vs._JDS.jpg",
  boxing: "https://upload.wikimedia.org/wikipedia/commons/2/2a/Boxing_Tournament_in_Aid_of_King_George%27s_Fund_For_Sailors_at_the_Royal_Naval_Air_Station%2C_Henstridge%2C_Somerset%2C_July_1945_A29806.jpg",
  cricket: "https://upload.wikimedia.org/wikipedia/commons/7/7a/Pollock_to_Hussey.jpg",
  tennis: "https://upload.wikimedia.org/wikipedia/commons/9/94/2013_Australian_Open_-_Guillaume_Rufin.jpg",
  golf: "https://upload.wikimedia.org/wikipedia/commons/6/6e/Golfer_swing.jpg",

  // Generic multi-sport venue shot for "other" and anything unmapped.
  other: "https://upload.wikimedia.org/wikipedia/commons/thumb/6/60/Olympic_Park%2C_London%2C_16_April_2012.jpg/1280px-Olympic_Park%2C_London%2C_16_April_2012.jpg",

};

const DEFAULT_IMAGE = IMAGES.other!;

export function sportsBackgroundImage(category: string): string {

  return IMAGES[category.toLowerCase()] ?? DEFAULT_IMAGE;

}
