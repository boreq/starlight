package protocol

// All numeric replies defined in RFC 2812.
type Numeric int

const (
	// Welcome to the Internet Relay Network <nick>!<user>@<host>
	RPL_WELCOME Numeric = 1

	// Your host is <servername>, running version <ver>
	RPL_YOURHOST Numeric = 2

	// This server was created <date>
	RPL_CREATED Numeric = 3

	// <servername> <version> <available user modes> <available channel modes>
	RPL_MYINFO Numeric = 4

	// Try server <server name>, port <port number>
	RPL_BOUNCE Numeric = 5

	// :*1<reply> *( " " <reply> )
	RPL_USERHOST Numeric = 302

	// :*1<nick> *(  " <nick> )"
	RPL_ISON Numeric = 303

	// "<nick> :<away message>"
	RPL_AWAY Numeric = 301

	// :You are no longer marked as being away
	RPL_UNAWAY Numeric = 305

	// :You have been marked as being away
	RPL_NOWAWAY Numeric = 306

	// <nick> <user> <host> * :<real name>
	RPL_WHOISUSER Numeric = 311

	// <nick> <server> :<server info>
	RPL_WHOISSERVER Numeric = 312

	// <nick> :is an IRC operator
	RPL_WHOISOPERATOR Numeric = 313

	// <nick> <integer> :seconds idle
	RPL_WHOISIDLE Numeric = 317

	// <nick> :End of WHOIS list
	RPL_ENDOFWHOIS Numeric = 318

	// <nick> :*( ( "@" / "+" ) <channel> " " )
	RPL_WHOISCHANNELS Numeric = 319

	// <nick> <user> <host> * :<real name>
	RPL_WHOWASUSER Numeric = 314

	// <nick> :End of WHOWAS
	RPL_ENDOFWHOWAS Numeric = 369

	// Obsolete. Not used.
	RPL_LISTSTART Numeric = 321

	// <channel> <// visible> :<topic>
	RPL_LIST Numeric = 322

	// :End of LIST
	RPL_LISTEND Numeric = 323

	// <channel> <nickname>
	RPL_UNIQOPIS Numeric = 325

	// <channel> <mode> <mode params>
	RPL_CHANNELMODEIS Numeric = 324

	// <channel> :No topic is set
	RPL_NOTOPIC Numeric = 331

	// <channel> :<topic>
	RPL_TOPIC Numeric = 332

	// <channel> <nick>
	RPL_INVITING Numeric = 341

	// <user> :Summoning user to IRC
	RPL_SUMMONING Numeric = 342

	// <channel> <invitemask>
	RPL_INVITELIST Numeric = 346

	// <channel> :End of channel invite list
	RPL_ENDOFINVITELIST Numeric = 347

	// <channel> <exceptionmask>
	RPL_EXCEPTLIST Numeric = 348

	// <channel> :End of channel exception list
	RPL_ENDOFEXCEPTLIST Numeric = 349

	// <version>.<debuglevel> <server> :<comments>
	RPL_VERSION Numeric = 351

	//"<channel> <user> <host> <server> <nick>
	//( "H" / "G" > ["*"] [ ( "@" / "+" ) ]
	//:<hopcount> <real name>"
	RPL_WHOREPLY Numeric = 352

	// <name> :End of WHO list
	RPL_ENDOFWHO Numeric = 315

	// "( "=" / "*" / "@" ) <channel>
	// :[ "@" / "+" ] <nick> *( " " [ "@" / "+" ] <nick> )
	// - "@" is used for secret channels, "*" for private
	// channels, and "=" for others (public channels).
	RPL_NAMREPLY Numeric = 353

	// <channel> :End of NAMES list
	RPL_ENDOFNAMES Numeric = 366

	// <mask> <server> :<hopcount> <server info>
	RPL_LINKS Numeric = 364

	// <mask> :End of LINKS list
	RPL_ENDOFLINKS Numeric = 365

	// <channel> <banmask>
	RPL_BANLIST Numeric = 367

	// <channel> :End of channel ban list
	RPL_ENDOFBANLIST Numeric = 368

	// :<string>
	RPL_INFO Numeric = 371

	// :End of INFO list
	RPL_ENDOFINFO Numeric = 374

	// :- <server> Message of the day -
	RPL_MOTDSTART Numeric = 375

	// :- <text>
	RPL_MOTD Numeric = 372

	// :End of MOTD command
	RPL_ENDOFMOTD Numeric = 376

	// :You are now an IRC operator
	RPL_YOUREOPER Numeric = 381

	// <config file> :Rehashing
	RPL_REHASHING Numeric = 382

	// You are service <servicename>
	RPL_YOURESERVICE Numeric = 383

	// <server> :<string showing server's local time>
	RPL_TIME Numeric = 391

	// :UserID   Terminal  Host
	RPL_USERSSTART Numeric = 392

	// :<username> <ttyline> <hostname>
	RPL_USERS Numeric = 393

	// :End of users
	RPL_ENDOFUSERS Numeric = 394

	// :Nobody logged in
	RPL_NOUSERS Numeric = 395

	// Link <version & debug level> <destination>
	// <next server> V<protocol version>
	// <link uptime in seconds> <backstream sendq>
	// <upstream sendq>
	RPL_TRACELINK Numeric = 200

	// Try. <class> <server>
	RPL_TRACECONNECTING Numeric = 201

	// H.S. <class> <server>
	RPL_TRACEHANDSHAKE Numeric = 202

	// ???? <class> [<client IP address in dot form>]
	RPL_TRACEUNKNOWN Numeric = 203

	// Oper <class> <nick>
	RPL_TRACEOPERATOR Numeric = 204

	// User <class> <nick>
	RPL_TRACEUSER Numeric = 205

	// Serv <class> <int>S <int>C <server>
	// <nick!user|*!*>@<host|server> V<protocol version>
	RPL_TRACESERVER Numeric = 206

	// Service <class> <name> <type> <active type>
	RPL_TRACESERVICE Numeric = 207

	// <newtype> 0 <client name>
	RPL_TRACENEWTYPE Numeric = 208

	// Class <class> <count>
	RPL_TRACECLASS Numeric = 209

	// Unused.
	RPL_TRACERECONNECT Numeric = 210

	// File <logfile> <debug level>
	RPL_TRACELOG Numeric = 261

	// <server name> <version & debug level> :End of TRACE
	RPL_TRACEEND Numeric = 262

	// <linkname> <sendq> <sent messages>
	// <sent Kbytes> <received messages>
	// <received Kbytes> <time open>
	RPL_STATSLINKINFO Numeric = 211

	// <command> <count> <byte count> <remote count>
	RPL_STATSCOMMANDS Numeric = 212

	// <stats letter> :End of STATS report
	RPL_ENDOFSTATS Numeric = 219

	// :Server Up %d days %d:%02d:%02d
	RPL_STATSUPTIME Numeric = 242

	// O <hostmask> * <name>
	RPL_STATSOLINE Numeric = 243

	// <user mode string>
	RPL_UMODEIS Numeric = 221

	// <name> <server> <mask> <type> <hopcount> <info>
	RPL_SERVLIST Numeric = 234

	// <mask> <type> :End of service listing
	RPL_SERVLISTEND Numeric = 235

	// :There are <integer> users and <integer>
	// services on <integer> servers
	RPL_LUSERCLIENT Numeric = 251

	// <integer> :operator(s) online
	RPL_LUSEROP Numeric = 252

	// <integer> :unknown connection(s)
	RPL_LUSERUNKNOWN Numeric = 253

	// <integer> :channels formed
	RPL_LUSERCHANNELS Numeric = 254

	// :I have <integer> clients and <integer>
	//  servers
	RPL_LUSERME Numeric = 255

	// <server> :Administrative info
	RPL_ADMINME Numeric = 256

	// :<admin info>
	RPL_ADMINLOC1 Numeric = 257

	// :<admin info>
	RPL_ADMINLOC2 Numeric = 258

	// :<admin info>
	RPL_ADMINEMAIL Numeric = 259

	// <command> :Please wait a while and try again.
	RPL_TRYAGAIN Numeric = 263

	// <nickname> :No such nick/channel
	ERR_NOSUCHNICK Numeric = 401

	// <server name> :No such server
	ERR_NOSUCHSERVER Numeric = 402

	// <channel name> :No such channel
	ERR_NOSUCHCHANNEL Numeric = 403

	// <channel name> :Cannot send to channel
	ERR_CANNOTSENDTOCHAN Numeric = 404

	// <channel name> :You have joined too many channels
	ERR_TOOMANYCHANNELS Numeric = 405

	// <nickname> :There was no such nickname
	ERR_WASNOSUCHNICK Numeric = 406

	// <target> :<error code> recipients. <abort message>
	ERR_TOOMANYTARGETS Numeric = 407

	// <service name> :No such service
	ERR_NOSUCHSERVICE Numeric = 408

	// :No origin specified
	ERR_NOORIGIN Numeric = 409

	// :No recipient given (<command>)
	ERR_NORECIPIENT Numeric = 411

	// :No text to send
	ERR_NOTEXTTOSEND Numeric = 412

	// <mask> :No toplevel domain specified
	ERR_NOTOPLEVEL Numeric = 413

	// <mask> :Wildcard in toplevel domain
	ERR_WILDTOPLEVEL Numeric = 414

	// <mask> :Bad Server/host mask
	ERR_BADMASK Numeric = 415

	// <command> :Unknown command
	ERR_UNKNOWNCOMMAND Numeric = 421

	// :MOTD File is missing
	ERR_NOMOTD Numeric = 422

	// <server> :No administrative info available
	ERR_NOADMININFO Numeric = 423

	// :File error doing <file op> on <file>
	ERR_FILEERROR Numeric = 424

	// :No nickname given
	ERR_NONICKNAMEGIVEN Numeric = 431

	// <nick> :Erroneous nickname
	ERR_ERRONEUSNICKNAME Numeric = 432

	// <nick> :Nickname is already in use
	ERR_NICKNAMEINUSE Numeric = 433

	// <nick> :Nickname collision KILL from <user>@<host>
	ERR_NICKCOLLISION Numeric = 436

	// <nick/channel> :Nick/channel is temporarily unavailable
	ERR_UNAVAILRESOURCE Numeric = 437

	// <nick> <channel> :They aren't on that channel
	ERR_USERNOTINCHANNEL Numeric = 441

	// <channel> :You're not on that channel
	ERR_NOTONCHANNEL Numeric = 442

	// <user> <channel> :is already on channel
	ERR_USERONCHANNEL Numeric = 443

	// <user> :User not logged in
	ERR_NOLOGIN Numeric = 444

	// :SUMMON has been disabled
	ERR_SUMMONDISABLED Numeric = 445

	// :USERS has been disabled
	ERR_USERSDISABLED Numeric = 446

	// :You have not registered
	ERR_NOTREGISTERED Numeric = 451

	// <command> :Not enough parameters
	ERR_NEEDMOREPARAMS Numeric = 461

	// :Unauthorized command (already registered)
	ERR_ALREADYREGISTRED Numeric = 462

	// :Your host isn't among the privileged
	ERR_NOPERMFORHOST Numeric = 463

	// :Password incorrect
	ERR_PASSWDMISMATCH Numeric = 464

	// :You are banned from this server
	ERR_YOUREBANNEDCREEP Numeric = 465

	ERR_YOUWILLBEBANNED Numeric = 466

	// <channel> :Channel key already set
	ERR_KEYSET Numeric = 467

	// <channel> :Cannot join channel (+l)
	ERR_CHANNELISFULL Numeric = 471

	// <char> :is unknown mode char to me for <channel>
	ERR_UNKNOWNMODE Numeric = 472

	// <channel> :Cannot join channel (+i)
	ERR_INVITEONLYCHAN Numeric = 473

	// <channel> :Cannot join channel (+b)
	ERR_BANNEDFROMCHAN Numeric = 474

	// <channel> :Cannot join channel (+k)
	ERR_BADCHANNELKEY Numeric = 475

	// <channel> :Bad Channel Mask
	ERR_BADCHANMASK Numeric = 476

	// <channel> :Channel doesn't support modes
	ERR_NOCHANMODES Numeric = 477

	// <channel> <char> :Channel list is full
	ERR_BANLISTFULL Numeric = 478

	// :Permission Denied- You're not an IRC operator
	ERR_NOPRIVILEGES Numeric = 481

	// <channel> :You're not channel operator
	ERR_CHANOPRIVSNEEDED Numeric = 482

	// :You can't kill a server!
	ERR_CANTKILLSERVER Numeric = 483

	// :Your connection is restricted!
	ERR_RESTRICTED Numeric = 484

	// :You're not the original channel operator
	ERR_UNIQOPPRIVSNEEDED Numeric = 485

	// :No O-lines for your host
	ERR_NOOPERHOST Numeric = 491

	// :Unknown MODE flag
	ERR_UMODEUNKNOWNFLAG Numeric = 501

	// :Cannot change mode for other users
	ERR_USERSDONTMATCH Numeric = 502
)
