package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	g810 "github.com/ticpu/go-g810"
)

type CpuStats struct {
	Total int
	Idle  int
}

func getCPUStats(cpuStatsOld *[12]CpuStats, keyMap *[12]g810.KeyValue) []g810.KeyValue {
	var cpuNo int
	var cpuStatsNew [12]CpuStats
	var currentUsage = g810.KeyColor{Red: 0, Green: 0, Blue: 0}

	stat, err := os.Open("/proc/stat")

	if err != nil {
		log.Fatal(err)
	}
	defer stat.Close()

	stat.Seek(0, 0)
	scanner := bufio.NewScanner(stat)
	cpuStatsMatch := regexp.MustCompile("^cpu([0-9]+)")
	for scanner.Scan() {
		cpuStats := strings.Split(scanner.Text(), " ")
		m := cpuStatsMatch.FindStringSubmatch(cpuStats[0])
		if m != nil {
			cpuNo, _ = strconv.Atoi(m[1])
			if cpuNo > len(cpuStats) {
				log.Fatal("Too many CPUs")
				break
			}
			cpuIdle, _ := strconv.Atoi(cpuStats[4])
			cpuTotal := 0
			for _, statCol := range cpuStats[1:] {
				if statColInt, err := strconv.Atoi(statCol); err == nil {
					cpuTotal += statColInt
				}
				cpuStatsNew[cpuNo].Total = cpuTotal
				cpuStatsNew[cpuNo].Idle = cpuIdle
			}
		}
	}
	cpuNo++

	for n := 0; n < cpuNo; n++ {
		currentUsage.Green = 255 - (uint8)((cpuStatsNew[n].Idle-cpuStatsOld[n].Idle)*255/(cpuStatsNew[n].Total-cpuStatsOld[n].Total))
		if currentUsage.Green > 215 {
			currentUsage.Red = (uint8)((currentUsage.Green - 215) * (215 / 40))
			currentUsage.Green = 255 - currentUsage.Red
		} else {
			currentUsage.Red = 0
			currentUsage.Green += 40
		}
		currentUsage.Green = (keyMap[n].Color.Green + currentUsage.Green) / 2
		keyMap[n] = g810.KeyValue{
			ID: g810.KeyF1 + (g810.Key)(n),
			Color: currentUsage,
		}
		cpuStatsOld[n].Idle = cpuStatsNew[n].Idle
		cpuStatsOld[n].Total = cpuStatsNew[n].Total
	}

	return keyMap[:]
}

func getCPUPercent(keyMapChan chan []g810.KeyValue) {
	var cpuStats [12]CpuStats
	var keyMap [12]g810.KeyValue

	for {
		keyMap := getCPUStats(&cpuStats, &keyMap)
		keyMapChan<-keyMap
		time.Sleep(200 * time.Millisecond)
	}
}

func main() {
	var n g810.Key
	var keyMap []g810.KeyValue
	keyMapChan := make(chan []g810.KeyValue)
	newChanges := false

	for n = g810.KeyA; n < 1028+26; n++ {
		keyMap = append(keyMap, g810.KeyValue{
			ID: n,
			Color: g810.KeyColor{Red: 95, Green: 0, Blue: 45},
		})
	}

	lk := g810.NewLedKeyboard()
	defer lk.Free()
	lk.Open()
	defer lk.Close()

	deviceInfo := lk.GetDeviceInfo()
	fmt.Printf(
		"Vendor: %s (0x%04X)\nProduct: %s (0x%04X)\nModel: %s\nS/N: %s\n",
		deviceInfo.Manufacturer, deviceInfo.VendorID,
		deviceInfo.Product, deviceInfo.ProductID,
		deviceInfo.KeyboardModel,
		deviceInfo.SerialNumber,
	)

	lk.SetAllKeys(g810.KeyColor{Red: 80, Green: 0, Blue: 120})
	lk.SetGroupKeys(g810.GroupLogo, g810.KeyColor{Red: 0, Green: 40, Blue: 40})
	lk.SetGroupKeys(g810.GroupFKeys, g810.KeyColor{Red: 0, Green: 40, Blue: 0})
	lk.SetGroupKeys(g810.GroupArrows, g810.KeyColor{Red: 0, Green: 120, Blue: 120})
	lk.SetKeys(keyMap)

	go getCPUPercent(keyMapChan)

	for {
		select {
		case keyMap = <-keyMapChan:
			newChanges = true
		default:
			time.Sleep(100 * time.Millisecond)
			if newChanges {
				newChanges = false
				if err := lk.SetKeys(keyMap); err != nil {
					log.Fatal(err)
				}
				if err := lk.Commit(); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
