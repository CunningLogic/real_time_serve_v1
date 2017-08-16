package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/alecthomas/log4go"
	"gopkg.in/gorp.v1"
	"strings"
	"time"
)

type DBManager struct {
  dbmap                 *gorp.DbMap

  iuavFlightDataQueue   chan IuavFlightData
}

func NewDBManager() *DBManager {
  db, err := sql.Open("mysql", GetConfigInstance().Database)
  checkError(err)

  err = db.Ping()
  checkError(err)

  db.SetMaxOpenConns(20)
  db.SetMaxIdleConns(11)

  dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
  dbmap.AddTableWithName(IuavFlightData{}, "iuav_flight_data")
  dbmap.AddTableWithName(OnlineInfo{}, "iuav_online_info").SetKeys(true, "Id")
  dbmap.AddTableWithName(ActiveDBInfo{}, "agro_active_info").SetKeys(true, "Id")

  return &DBManager{
    dbmap:               dbmap,
    iuavFlightDataQueue: make(chan IuavFlightData, 5000),
  }
}

func (dbMgr *DBManager) PushToIuavFlightData(data IuavFlightData) error {
  select {
    case dbMgr.iuavFlightDataQueue <- data:
    default:
      err := dbMgr.dbmap.Insert(data)
      if err != nil {
        fmt.Println("insert to iuav flight data failed", err)
        return err
      }
  }

  return nil
}

func (dbMgr *DBManager) StartDBSaver() {
  go dbMgr.IuavFlightDataDBSaver()
}

func (dbMgr *DBManager) getHardwareInfoBySn(sn string) (HardwareInfo, error) {
	var result HardwareInfo
	query := fmt.Sprintf("select uid, nickname, type, team_id, locked from %s where deleted = 0 and hardware_id = ?", IUAV_AGRO_TABLE_NAME)
	err := dbMgr.dbmap.SelectOne(&result, query, sn)
	if err != nil {
		return result, err
	}
	query = fmt.Sprintf("select realname from %s where deleted = 0 and uid = ?", IUAV_AGRO_FLYER)
	result.BossName, err = dbMgr.dbmap.SelectStr(query, result.BossID)

	return result, err
}

func (dbMgr *DBManager) getTeamNameById(teamid uint32) (string, string, error) {
  var teamInfo struct {
    Name    sql.NullString `db:"name"`
    Captain sql.NullString `db:"captain"`
  }
  query := fmt.Sprintf("select name, captain from %s where deleted = 0 and id = ?", IUAV_AGRO_TEAM)
  err := dbMgr.dbmap.SelectOne(&teamInfo, query, teamid)
  return teamInfo.Name.String, teamInfo.Captain.String, err
}

func (dbMgr *DBManager) getManagerUidsByHardwareSN(HardwareSN string) ([]string, error) {
  var managerUids []string
  query := fmt.Sprintf("select uid from %s where hardware_id = ?", IUAV_AGRO_MANAGER_DRONE)
  _, err := dbMgr.dbmap.Select(&managerUids, query, HardwareSN)
  return managerUids, err
}

func (dbMgr *DBManager) GetUserInfoByUid(uid uint64, bossID string) (string, uint8, error) {
  var flyerinfo struct {
    Realname  string `db:"realname"`
    Job_level uint8  `db:"job_level"`
  }
  query := fmt.Sprintf("select realname, job_level from %s where deleted = 0 and uid = ? and upper_uid = ?", IUAV_AGRO_FLYER)
  err := dbMgr.dbmap.SelectOne(&flyerinfo, query, uid, bossID)
  return flyerinfo.Realname, flyerinfo.Job_level, err
}

func (dbMgr *DBManager) GetDroneTodayWork(sn string) (uint32, error) {
  t := time.Now().Year()
  m := time.Now().Month()
  d := time.Now().Day()
  today := t*10000 + int(m)*100 + d
  query := fmt.Sprintf("select sum(work_area) from %s where product_sn = ? and create_date = ?", IUAV_AGRO_FLIGHT)
  sum, err := dbMgr.dbmap.SelectInt(query, sn, today)
  return uint32(sum), err
}

func (dbMgr *DBManager) RecordOnlineInfo(flyer FlyerInfo) error {
  var online OnlineInfo
  online.SN = flyer.HardwareSN
  online.UserID = flyer.UserID
  online.BossID = flyer.BossID
  online.Captain = flyer.Captain
  online.ServerInfo = "serverinfo"
  online.TimeStamp = time.Now().Unix()

  err := dbMgr.dbmap.Insert(online)
  if err != nil {
    fmt.Println("InsertDroneInfo failed", err, online)
  }

  return err
}

func (dbMgr *DBManager) UpdateDroneStatus(sn string, is_online uint8) error {
  //标明这架飞机已经在线
  if sn == "" {
    return nil
  }

  query := fmt.Sprintf("update %s set is_online = %d where hardware_id = '%s' and deleted = 0", IUAV_AGRO_TABLE_NAME, is_online, sn)
  _, err := dbMgr.dbmap.Exec(query)
  return err
}

func (s *Service) GetBossIDBySn(sn string) (string, error) {
	var uid string
	query := fmt.Sprintf("select uid from %s where hardware_id = ?", IUAV_AGRO_TABLE_NAME)
	err := s.dbMgr.dbmap.SelectOne(&uid, query, sn)
	return uid, err
}

func (s *Service) GetSuperDroneLockedInfo(uid string) (LockedInfo, error) {
	var result LockedInfo
	if uid != _SUPER_UID_ {
		return result, errors.New("not the super uid.")
	}
	var err error
	query := fmt.Sprintf("select count(id) from %s", IUAV_AGRO_TABLE_NAME)
	result.Drone_sum, err = s.dbMgr.dbmap.SelectInt(query)

	query = fmt.Sprintf("select hardware_id from %s where deleted = 0 and locked = 1", IUAV_AGRO_TABLE_NAME)
	_, err = s.dbMgr.dbmap.Select(&result.Locked_drones, query)
	return result, err
}

func (s *Service) GetDroneLockedInfo(uid string) (LockedInfo, error) {
	var result LockedInfo
	var err error
	accessibleDroneSNs := s.GetUserAccessibleDroneHardwareSNs(uid)
	if len(accessibleDroneSNs) > 0 {
		inClause := strings.Join(accessibleDroneSNs, `","`)
		query := fmt.Sprintf(`select count(id) from %s where hardware_id IN ("` + inClause + `")`, IUAV_AGRO_TABLE_NAME)
		result.Drone_sum, err = s.dbMgr.dbmap.SelectInt(query)

		query = fmt.Sprintf(`select hardware_id from %s where deleted = 0 and locked = 1 and hardware_id IN ("` + inClause + `")`, IUAV_AGRO_TABLE_NAME)
		_, err = s.dbMgr.dbmap.Select(&result.Locked_drones, query)
	} else {
		result.Drone_sum = 0
		result.Locked_drones = []string{}
	}
	return result, err
}

func (s *Service) GetUserTeamIds(uid string) ([]string) {
	var TeamIds []string
	query := fmt.Sprintf("select id from %s where captain = ?", IUAV_AGRO_TEAM)
  s.dbMgr.dbmap.Select(&TeamIds, query, uid)

	return TeamIds
}

func (s *Service) GetUserAccessibleDroneHardwareSNs(uid string) ([]string) {
	var ownedDroneHardwareSNs []string
	var managedDroneHardwareSNs []string
	var query string

	teamIds := s.GetUserTeamIds(uid)

	if len(teamIds) > 0 {
		inClause := strings.Join(teamIds, `","`)
		query = fmt.Sprintf(`select hardware_id from %s where uid = ? OR team_id IN ("` + inClause + `")`, IUAV_AGRO_TABLE_NAME)
	} else {
		query = fmt.Sprintf(`select hardware_id from %s where uid = ?`, IUAV_AGRO_TABLE_NAME)
	}

  s.dbMgr.dbmap.Select(&ownedDroneHardwareSNs, query, uid)

	query = fmt.Sprintf("select hardware_id from %s where uid = ?", IUAV_AGRO_MANAGER_DRONE)
  s.dbMgr.dbmap.Select(&managedDroneHardwareSNs, query, uid)

	return append(ownedDroneHardwareSNs, managedDroneHardwareSNs...)
}

func (s *Service) DeleteOnlineInfo(info *OnlineInfo) error {
	_, err := s.dbMgr.dbmap.Delete(info)
	return err
}

func (s *Service) QueryOnlineDroneInfo(boss_id string) ([]string, error) {
	var drones []string
	query := fmt.Sprintf("select sn from %s where boss_id = %s", IUAV_ONLINE_INFO_TABLE_NAME, boss_id)
	_, err := s.dbMgr.dbmap.Select(&drones, query)
	return drones, err
}

func InsertToIuavFlightData(items []IuavFlightData) error {
  trans, err := service.dbMgr.dbmap.Begin()
  if err != nil {
    return err
  }

  for _, v := range items {
    trans.Insert(&v)
  }

  return trans.Commit()
}

func (dbMgr *DBManager) IuavFlightDataDBSaver() {
  dataset := make([]IuavFlightData, 0, 10)

  havedata := false

  for {
    select {
      case <- service.stop:
        log4go.Info("iuav_flight_data saver stop")
        return
      default:
    }

    for {
      select {
      case data := <- dbMgr.iuavFlightDataQueue:
        dataset = append(dataset, data)
        if len(dataset) > 9 {
          havedata = true
          goto INSERTTODB
        }
      case <- time.After(time.Second * 2):
        if len(dataset) > 0 {
          havedata = true
          goto INSERTTODB
        }
      }
    }

    INSERTTODB:
    if havedata {
      err := InsertToIuavFlightData(dataset)
      if err != nil {
        fmt.Println("iuav_flight_data insert failed", err)
      }

      dataset = dataset[:0]
    }
  }
}
