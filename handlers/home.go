package handlers

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    
    "github.com/Jake-Schuler/ORC-MatchMaker/services"
)

var isSchedulePublic = false

func HomeHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !isSchedulePublic {
            c.HTML(200, "index.tmpl", gin.H{
                "title":            "ORC Match Maker",
                "isSchedulePublic": isSchedulePublic,
            })
        } else {
            c.HTML(200, "index.tmpl", gin.H{
                "title":            "ORC Match Maker",
                "matches":          services.ParseMatchSchedule(),
                "isSchedulePublic": isSchedulePublic,
            })
        }
    }
}

func GetSchedulePublic() bool {
    return isSchedulePublic
}

func SetSchedulePublic(value bool) {
    isSchedulePublic = value
}