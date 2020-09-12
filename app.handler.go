package road

import (
	"github.com/lixianmin/road/component"
)

/********************************************************************
created:    2020-08-29
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

var (
	handlers = make(map[string]*component.Handler) // all handler method
)